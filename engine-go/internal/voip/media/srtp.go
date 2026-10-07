package media

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
)

type SrtpErrorType string

const (
	SrtpErrPacketTooShort SrtpErrorType = "packet_too_short"
	SrtpErrAuthFailed     SrtpErrorType = "auth_failed"
	SrtpErrEncryption     SrtpErrorType = "encryption"
	SrtpErrDecryption     SrtpErrorType = "decryption"
	SrtpErrReplay         SrtpErrorType = "replay"
)

// srtpReplayWindowSize é a janela anti-replay (RFC 3711 §3.3.2) em pacotes.
const srtpReplayWindowSize = 64

type SrtpError struct {
	Type SrtpErrorType
	Msg  string
}

func (e *SrtpError) Error() string { return fmt.Sprintf("srtp %s: %s", e.Type, e.Msg) }

type SrtpContext struct {
	mu          sync.Mutex
	sessionKey  []byte
	sessionSalt []byte
	authKey     []byte
	roc         uint32
	lastSeq     uint16
	initialized bool
	authTagLen  int

	replayHighest uint64
	replayWindow  uint64
	replaySeen    bool
}

func NewSrtpContext(keying core.SrtpKeyingMaterial, authTagLen int) (*SrtpContext, error) {
	if authTagLen <= 0 {
		authTagLen = core.SRTPAuthTagLen
	}
	sk, err := deriveSrtpKey(keying.MasterKey, keying.MasterSalt, core.SRTPLabelEncryption, 16)
	if err != nil {
		return nil, err
	}
	ak, err := deriveSrtpKey(keying.MasterKey, keying.MasterSalt, core.SRTPLabelAuth, 20)
	if err != nil {
		return nil, err
	}
	ss, err := deriveSrtpKey(keying.MasterKey, keying.MasterSalt, core.SRTPLabelSalt, 14)
	if err != nil {
		return nil, err
	}
	return &SrtpContext{
		sessionKey:  sk,
		sessionSalt: ss,
		authKey:     ak,
		authTagLen:  authTagLen,
	}, nil
}

func (c *SrtpContext) SetAuthKeying(keying core.SrtpKeyingMaterial) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ak, err := deriveSrtpKey(keying.MasterKey, keying.MasterSalt, core.SRTPLabelAuth, 20)
	if err != nil {
		return err
	}
	c.authKey = ak
	return nil
}

func (c *SrtpContext) Protect(packet *RtpPacket) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updateRoc(packet.Header.SequenceNumber)
	index := c.packetIndex(packet.Header.SequenceNumber)

	headerSize := packet.Header.Size()
	output := make([]byte, headerSize+len(packet.Payload)+c.authTagLen)

	if _, err := packet.Header.Encode(output); err != nil {
		return nil, &SrtpError{SrtpErrEncryption, err.Error()}
	}

	iv := c.generateIV(packet.Header.Ssrc, index)
	if err := aesCtrXor(c.sessionKey, iv, packet.Payload, output[headerSize:headerSize+len(packet.Payload)]); err != nil {
		return nil, &SrtpError{SrtpErrEncryption, err.Error()}
	}

	if c.authTagLen > 0 {
		authData := output[:headerSize+len(packet.Payload)]
		tag := c.computeAuthTag(authData, c.roc, c.authTagLen)
		copy(output[headerSize+len(packet.Payload):], tag)
	}

	return output, nil
}

// Unprotect valida e decifra um pacote SRTP recebido.
//
// A ordem importa: estima o ROC SEM alterar o estado, rejeita repetição, verifica a tag de
// autenticação e SÓ ENTÃO avança ROC e janela. Assim um pacote forjado (tag errada) nunca
// consegue dessincronizar o contador nem "queimar" um número de sequência legítimo.
func (c *SrtpContext) Unprotect(data []byte) (*RtpPacket, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(data) < 12 {
		return nil, &SrtpError{SrtpErrPacketTooShort, fmt.Sprintf("packet too short: %d bytes", len(data))}
	}

	header, err := DecodeRtpHeader(data)
	if err != nil {
		return nil, &SrtpError{SrtpErrDecryption, err.Error()}
	}
	headerSize := header.Size()
	payloadLen := len(data) - headerSize - c.authTagLen
	if payloadLen <= 0 {
		return nil, &SrtpError{SrtpErrPacketTooShort, fmt.Sprintf("no payload: %dB total, %dB header, auth=%d", len(data), headerSize, c.authTagLen)}
	}

	roc := c.estimateRoc(header.SequenceNumber)
	index := (uint64(roc) << 16) | uint64(header.SequenceNumber)
	if err := c.replayCheck(index); err != nil {
		return nil, err
	}
	if c.authTagLen > 0 {
		expected := c.computeAuthTag(data[:headerSize+payloadLen], roc, c.authTagLen)
		if !hmac.Equal(expected, data[headerSize+payloadLen:]) {
			return nil, &SrtpError{SrtpErrAuthFailed, fmt.Sprintf("auth tag mismatch for seq %d", header.SequenceNumber)}
		}
	}
	c.commitRoc(roc, header.SequenceNumber)
	c.replayUpdate(index)

	iv := c.generateIV(header.Ssrc, index)
	decrypted := make([]byte, payloadLen)
	if err := aesCtrXor(c.sessionKey, iv, data[headerSize:headerSize+payloadLen], decrypted); err != nil {
		return nil, &SrtpError{SrtpErrDecryption, err.Error()}
	}

	return &RtpPacket{Header: header, Payload: decrypted}, nil
}

func (c *SrtpContext) updateRoc(seq uint16) {
	if !c.initialized {
		c.lastSeq = seq
		c.initialized = true
		return
	}

	diff := int32(seq) - int32(c.lastSeq)
	if diff < -32768 {
		c.roc++
	}
	c.lastSeq = seq
}

func (c *SrtpContext) packetIndex(seq uint16) uint64 {
	return (uint64(c.roc) << 16) | uint64(seq)
}

// estimateRoc calcula o ROC provável de um pacote recebido, sem alterar o estado (RFC 3711
// §3.3.1). Um pacote ligeiramente atrasado logo após a virada de 65535→0 pertence ao ROC anterior.
func (c *SrtpContext) estimateRoc(seq uint16) uint32 {
	if !c.initialized {
		return c.roc
	}
	if c.lastSeq < 0x8000 {
		if int32(seq)-int32(c.lastSeq) > 0x8000 {
			return c.roc - 1
		}
		return c.roc
	}
	if int32(c.lastSeq)-int32(seq) > 0x8000 {
		return c.roc + 1
	}
	return c.roc
}

// commitRoc grava ROC e último seq, só depois de o pacote ter sido autenticado.
func (c *SrtpContext) commitRoc(v uint32, seq uint16) {
	if !c.initialized {
		c.lastSeq = seq
		c.initialized = true
		return
	}
	switch v {
	case c.roc:
		if seq > c.lastSeq {
			c.lastSeq = seq
		}
	case c.roc + 1:
		c.roc = v
		c.lastSeq = seq
	}
}

// replayCheck rejeita um índice já visto ou mais antigo que a janela (RFC 3711 §3.3.2).
func (c *SrtpContext) replayCheck(index uint64) error {
	if !c.replaySeen || index > c.replayHighest {
		return nil
	}
	delta := c.replayHighest - index
	if delta >= srtpReplayWindowSize {
		return &SrtpError{SrtpErrReplay, fmt.Sprintf("index %d older than replay window", index)}
	}
	if c.replayWindow&(1<<delta) != 0 {
		return &SrtpError{SrtpErrReplay, fmt.Sprintf("duplicate index %d", index)}
	}
	return nil
}

// replayUpdate marca o índice como recebido.
func (c *SrtpContext) replayUpdate(index uint64) {
	if !c.replaySeen {
		c.replaySeen = true
		c.replayHighest = index
		c.replayWindow = 1
		return
	}
	if index > c.replayHighest {
		delta := index - c.replayHighest
		if delta >= srtpReplayWindowSize {
			c.replayWindow = 1
		} else {
			c.replayWindow = c.replayWindow<<delta | 1
		}
		c.replayHighest = index
		return
	}
	c.replayWindow |= 1 << (c.replayHighest - index)
}

func (c *SrtpContext) generateIV(ssrc uint32, index uint64) []byte {
	iv := make([]byte, 16)
	copy(iv, c.sessionSalt[:14])

	var ssrcBuf [4]byte
	binary.BigEndian.PutUint32(ssrcBuf[:], ssrc)
	for i := 0; i < 4; i++ {
		iv[4+i] ^= ssrcBuf[i]
	}

	var idxBuf [8]byte
	binary.BigEndian.PutUint64(idxBuf[:], index)
	for i := 0; i < 6; i++ {
		iv[8+i] ^= idxBuf[2+i]
	}

	return iv
}

func (c *SrtpContext) computeAuthTag(data []byte, roc uint32, tagLen int) []byte {
	mac := hmac.New(sha1.New, c.authKey)
	mac.Write(data)
	var rocBuf [4]byte
	binary.BigEndian.PutUint32(rocBuf[:], roc)
	mac.Write(rocBuf[:])
	sum := mac.Sum(nil)
	return sum[:tagLen]
}

// SrtpSession protege e desprotege os fluxos de uma chamada com as MESMAS chaves, mas com estado de
// sequência, ROC e janela anti-replay SEPARADO por SSRC (RFC 3711: o contexto criptográfico é por fluxo).
// Áudio e vídeo contam a sequência de formas independentes; num estado único o vídeo empurrava a janela
// do áudio e o derrubava.
type SrtpSession struct {
	mu          sync.Mutex
	sendKeying  core.SrtpKeyingMaterial
	recvKeying  core.SrtpKeyingMaterial
	sendAuthLen int
	recvAuthLen int
	sendCtx     map[uint32]*SrtpContext
	recvCtx     map[uint32]*SrtpContext
	sendAuthKM  *core.SrtpKeyingMaterial
}

func NewSrtpSession(sendKey, recvKey core.SrtpKeyingMaterial, sendAuthLen, recvAuthLen int) (*SrtpSession, error) {
	// Valida as chaves já na criação (e deixa o erro como antes), criando o contexto "sem SSRC".
	if _, err := NewSrtpContext(sendKey, sendAuthLen); err != nil {
		return nil, err
	}
	if _, err := NewSrtpContext(recvKey, recvAuthLen); err != nil {
		return nil, err
	}
	return &SrtpSession{
		sendKeying: sendKey, recvKeying: recvKey, sendAuthLen: sendAuthLen, recvAuthLen: recvAuthLen,
		sendCtx: map[uint32]*SrtpContext{}, recvCtx: map[uint32]*SrtpContext{},
	}, nil
}

func (s *SrtpSession) sendFor(ssrc uint32) (*SrtpContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.sendCtx[ssrc]; ok {
		return c, nil
	}
	c, err := NewSrtpContext(s.sendKeying, s.sendAuthLen)
	if err != nil {
		return nil, err
	}
	if s.sendAuthKM != nil {
		if err := c.SetAuthKeying(*s.sendAuthKM); err != nil {
			return nil, err
		}
	}
	s.sendCtx[ssrc] = c
	return c, nil
}

func (s *SrtpSession) recvFor(ssrc uint32) (*SrtpContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.recvCtx[ssrc]; ok {
		return c, nil
	}
	c, err := NewSrtpContext(s.recvKeying, s.recvAuthLen)
	if err != nil {
		return nil, err
	}
	s.recvCtx[ssrc] = c
	return c, nil
}

func (s *SrtpSession) Protect(packet *RtpPacket) ([]byte, error) {
	c, err := s.sendFor(packet.Header.Ssrc)
	if err != nil {
		return nil, &SrtpError{SrtpErrEncryption, err.Error()}
	}
	return c.Protect(packet)
}

// Unprotect lê o SSRC do cabeçalho (bytes 8-11, ainda em claro) para escolher o contexto do fluxo.
func (s *SrtpSession) Unprotect(data []byte) (*RtpPacket, error) {
	if len(data) < 12 {
		return nil, &SrtpError{SrtpErrPacketTooShort, fmt.Sprintf("packet too short: %d bytes", len(data))}
	}
	ssrc := binary.BigEndian.Uint32(data[8:12])
	c, err := s.recvFor(ssrc)
	if err != nil {
		return nil, &SrtpError{SrtpErrDecryption, err.Error()}
	}
	return c.Unprotect(data)
}

func (s *SrtpSession) SetSendAuthKeying(keying core.SrtpKeyingMaterial) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendAuthKM = &keying
	for _, c := range s.sendCtx {
		if err := c.SetAuthKeying(keying); err != nil {
			return err
		}
	}
	return nil
}

func deriveSrtpKey(masterKey, masterSalt []byte, label byte, length int) ([]byte, error) {
	iv := make([]byte, 16)
	copy(iv, masterSalt[:14])
	iv[7] ^= label

	out := make([]byte, length)
	if err := aesCtrXor(masterKey, iv, make([]byte, length), out); err != nil {
		return nil, err
	}
	return out, nil
}

func aesCtrXor(key, iv, src, dst []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	cipher.NewCTR(block, iv).XORKeyStream(dst, src)
	return nil
}

package media

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"errors"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
)

// SRTCP do WhatsApp: as chaves-mestras por JID são as mesmas do SRTP; só mudam os rótulos da derivação
// (RFC 3711: 3 cifra, 4 autenticação, 5 sal) e a tag de autenticação, de 10 bytes. Formato e comportamento
// portados de purpshell/meowcaller (srtp/e2e.go, MIT, Rajeh Taher). Procedência em ../NOTICE.md.

const (
	srtcpLabelEncryption byte = 0x03
	srtcpLabelAuth       byte = 0x04
	srtcpLabelSalt       byte = 0x05

	// SrtcpAuthTagLen é a tag HMAC-SHA1 truncada de um pacote SRTCP.
	SrtcpAuthTagLen = 10
	// SrtcpTrailerLen é a palavra de índice (4) mais a tag.
	SrtcpTrailerLen = 4 + SrtcpAuthTagLen
	// rtcpClearHeaderLen são os 8 bytes (cabeçalho + SSRC do emissor) que o SRTCP deixa em claro.
	rtcpClearHeaderLen = 8
)

var ErrSrtcpShort = errors.New("srtcp: pacote curto demais")

// SrtcpKeys são as chaves de sessão do SRTCP de um participante.
type SrtcpKeys struct {
	cipherKey []byte
	salt      []byte
	authKey   []byte
}

// DeriveSrtcpKeys parte da chave-mestra por JID (a mesma do SRTP) e devolve as chaves do SRTCP.
func DeriveSrtcpKeys(keying core.SrtpKeyingMaterial) (*SrtcpKeys, error) {
	ck, err := deriveSrtpKey(keying.MasterKey, keying.MasterSalt, srtcpLabelEncryption, 16)
	if err != nil {
		return nil, err
	}
	ak, err := deriveSrtpKey(keying.MasterKey, keying.MasterSalt, srtcpLabelAuth, 20)
	if err != nil {
		return nil, err
	}
	sk, err := deriveSrtpKey(keying.MasterKey, keying.MasterSalt, srtcpLabelSalt, 14)
	if err != nil {
		return nil, err
	}
	return &SrtcpKeys{cipherKey: ck, authKey: ak, salt: sk}, nil
}

// srtcpIV: sal alinhado em 16 bytes, SSRC em XOR nos bytes 4-7 e o índice de 48 bits nos bytes 8-13.
func srtcpIV(salt []byte, ssrc, index uint32) []byte {
	iv := make([]byte, 16)
	copy(iv, salt)
	var s [4]byte
	binary.BigEndian.PutUint32(s[:], ssrc)
	for i := range s {
		iv[4+i] ^= s[i]
	}
	var idx [8]byte
	binary.BigEndian.PutUint64(idx[:], uint64(index))
	for i := 0; i < 6; i++ {
		iv[8+i] ^= idx[2+i]
	}
	return iv
}

func (k *SrtcpKeys) mac(data []byte) []byte {
	m := hmac.New(sha1.New, k.authKey)
	m.Write(data)
	return m.Sum(nil)[:SrtcpAuthTagLen]
}

// Protect cifra e autentica um pacote RTCP. index é o contador SRTCP do emissor (31 bits, crescente).
func (k *SrtcpKeys) Protect(senderSsrc, index uint32, rtcp []byte) ([]byte, error) {
	split := min(len(rtcp), rtcpClearHeaderLen)
	out := append([]byte(nil), rtcp[:split]...)
	body := make([]byte, len(rtcp)-split)
	if err := aesCtrXor(k.cipherKey, srtcpIV(k.salt, senderSsrc, index), rtcp[split:], body); err != nil {
		return nil, err
	}
	out = append(out, body...)
	out = binary.BigEndian.AppendUint32(out, 0x80000000|index)
	return append(out, k.mac(out)...), nil
}

// Unprotect autentica e decifra um pacote SRTCP; devolve o RTCP em claro e o índice.
func (k *SrtcpKeys) Unprotect(senderSsrc uint32, packet []byte) ([]byte, uint32, error) {
	if len(packet) < rtcpClearHeaderLen+SrtcpTrailerLen {
		return nil, 0, ErrSrtcpShort
	}
	tagStart := len(packet) - SrtcpAuthTagLen
	if !hmac.Equal(packet[tagStart:], k.mac(packet[:tagStart])) {
		return nil, 0, &SrtpError{SrtpErrAuthFailed, "srtcp tag"}
	}
	indexStart := tagStart - 4
	index := binary.BigEndian.Uint32(packet[indexStart:tagStart]) & 0x7fffffff
	body := make([]byte, indexStart-rtcpClearHeaderLen)
	if err := aesCtrXor(k.cipherKey, srtcpIV(k.salt, senderSsrc, index), packet[rtcpClearHeaderLen:indexStart], body); err != nil {
		return nil, 0, err
	}
	return append(append([]byte(nil), packet[:rtcpClearHeaderLen]...), body...), index, nil
}

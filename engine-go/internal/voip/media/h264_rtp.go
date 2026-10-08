package media

import "encoding/binary"

// Cabeçalho RTP de vídeo do WhatsApp: extensão 0xDEBE com metadados por pacote.
//
// Formato e vetores de teste portados de purpshell/meowcaller (rtp/rtp.go, MIT, Rajeh Taher). O
// `fabriciosprj/WaCalls-Video` descobriu que, sem esta extensão com o tipo de quadro, o WhatsApp
// nunca trata um quadro como keyframe. Procedência em ../NOTICE.md.

const (
	// PayloadTypeH264 é o payload type do vídeo H.264 do WhatsApp.
	PayloadTypeH264 uint8 = 97
	// WhatsappRtpExtensionProfile é o perfil da extensão (o mesmo do áudio).
	WhatsappRtpExtensionProfile uint16 = 0xdebe
	// VideoFrameInfoIDR e VideoFrameInfoDelta marcam o tipo do quadro em MediaFrameInfo.
	VideoFrameInfoIDR   uint8 = 0x08
	VideoFrameInfoDelta uint8 = 0x20
	// VideoSlotWord separa o SSRC de vídeo do de áudio (slot 0) na derivação HKDF.
	VideoSlotWord uint32 = 2
	// VideoClockRate é o relógio RTP do vídeo.
	VideoClockRate = 90000
)

// VideoRtpExtension são os metadados de vídeo de cada pacote.
type VideoRtpExtension struct {
	MediaFrameInfo    uint8
	FrameNumber       *uint16
	InitialBandwidth  uint16
	ShortOffset       int16
	TransportSequence uint16
}

// DisplayOrientation devolve a rotação CVO como quartos de volta no sentido horário.
func (e *VideoRtpExtension) DisplayOrientation() int { return int(e.MediaFrameInfo & 0x03) }

// Encode serializa o bloco de elementos de 1 byte (ids 3, 5, 6 e 9), alinhado a 4 bytes.
func (e *VideoRtpExtension) Encode() []byte {
	frameInfoLength := 1
	if e.FrameNumber != nil {
		frameInfoLength = 3
	}
	ext := make([]byte, 0, 16)
	ext = append(ext, 0x30|byte(frameInfoLength-1), e.MediaFrameInfo)
	if e.FrameNumber != nil {
		ext = binary.BigEndian.AppendUint16(ext, *e.FrameNumber)
	}
	ext = append(ext, 0x51)
	ext = binary.BigEndian.AppendUint16(ext, e.InitialBandwidth)
	ext = append(ext, 0x61)
	ext = binary.BigEndian.AppendUint16(ext, uint16(e.ShortOffset))
	ext = append(ext, 0x91)
	ext = binary.BigEndian.AppendUint16(ext, e.TransportSequence)
	for len(ext)%4 != 0 {
		ext = append(ext, 0)
	}
	return ext
}

// ParseVideoRtpExtension lê o bloco de um cabeçalho decodificado. ok=false se não for a extensão de
// vídeo do WhatsApp (perfil diferente, ou sem MediaFrameInfo e ShortOffset).
func ParseVideoRtpExtension(h *RtpHeader) (*VideoRtpExtension, bool) {
	if h == nil || !h.Extension || h.ExtensionProfile != WhatsappRtpExtensionProfile {
		return nil, false
	}
	ext := h.ExtensionData
	parsed := &VideoRtpExtension{}
	var hasFrameInfo, hasShortOffset bool
	for offset := 0; offset < len(ext); {
		header := ext[offset]
		offset++
		if header == 0 {
			continue
		}
		id := header >> 4
		length := int(header&0x0f) + 1
		if id == 15 || offset+length > len(ext) {
			return nil, false
		}
		value := ext[offset : offset+length]
		offset += length
		switch id {
		case 3:
			if length != 1 && length != 3 {
				return nil, false
			}
			parsed.MediaFrameInfo = value[0]
			if length == 3 {
				n := binary.BigEndian.Uint16(value[1:3])
				parsed.FrameNumber = &n
			}
			hasFrameInfo = true
		case 5:
			if length != 2 {
				return nil, false
			}
			parsed.InitialBandwidth = binary.BigEndian.Uint16(value)
		case 6:
			if length != 2 {
				return nil, false
			}
			parsed.ShortOffset = int16(binary.BigEndian.Uint16(value))
			hasShortOffset = true
		case 9:
			if length != 2 {
				return nil, false
			}
			parsed.TransportSequence = binary.BigEndian.Uint16(value)
		}
	}
	if !hasFrameInfo || !hasShortOffset {
		return nil, false
	}
	return parsed, true
}

// VideoRtpStream numera os pacotes de vídeo de uma chamada: sequência, sequência de transporte, número
// do quadro (só no 1º pacote de cada AU) e um único timestamp por AU.
type VideoRtpStream struct {
	ssrc              uint32
	seq               uint16
	timestamp         uint32
	tsStride          uint32
	transportSequence uint16
	frameNumber       uint16
	firstPacket       bool
}

// NewVideoRtpStream cria o numerador; tsStride é o avanço do timestamp por quadro (90000/fps).
func NewVideoRtpStream(ssrc, tsStride uint32) *VideoRtpStream {
	return &VideoRtpStream{ssrc: ssrc, seq: 1, tsStride: tsStride, frameNumber: 1, firstPacket: true}
}

// SetTimestampStride muda o avanço do timestamp a partir do próximo quadro (duração real do anterior).
func (s *VideoRtpStream) SetTimestampStride(stride uint32) {
	if stride != 0 {
		s.tsStride = stride
	}
}

// NextPacket devolve o cabeçalho do próximo pacote e a extensão que o acompanha.
func (s *VideoRtpStream) NextPacket(lastInAccessUnit bool, mediaFrameInfo uint8) (*RtpHeader, *VideoRtpExtension) {
	var frameNumber *uint16
	if s.firstPacket {
		v := s.frameNumber
		frameNumber = &v
	}
	ext := &VideoRtpExtension{MediaFrameInfo: mediaFrameInfo, FrameNumber: frameNumber, TransportSequence: s.transportSequence}
	body := ext.Encode()
	h := NewRtpHeader(PayloadTypeH264, s.seq, s.timestamp, s.ssrc)
	h.Marker = lastInAccessUnit
	h.Extension = true
	h.ExtensionProfile = WhatsappRtpExtensionProfile
	h.ExtensionData = body
	s.seq++
	s.transportSequence++
	if lastInAccessUnit {
		s.timestamp += s.tsStride
		s.frameNumber++
		s.firstPacket = true
	} else {
		s.firstPacket = false
	}
	return h, ext
}

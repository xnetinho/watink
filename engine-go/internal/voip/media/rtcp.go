package media

import "encoding/binary"

// RTCP do WhatsApp para vídeo: Sender Report + SDES periódicos (sem eles o vídeo do chamador não passa a
// fluir) e PLI/FIR para pedir quadro-chave. Formato e vetores portados de purpshell/meowcaller
// (rtp/rtcp.go, MIT, Rajeh Taher). Procedência em ../NOTICE.md.

const (
	rtcpPtSr   uint8 = 200
	rtcpPtSdes uint8 = 202
	rtcpPtPsfb uint8 = 206

	ntpUnixOffsetSecs uint64 = 2208988800

	// RtcpCnameLen é o tamanho do CNAME no formato aleatório do WhatsApp ("xxxxx@pjxxxxxx.org").
	RtcpCnameLen = 18
)

// RtcpSenderStats são os contadores do Sender Report.
type RtcpSenderStats struct {
	PacketsSent  uint32
	OctetsSent   uint32
	RtpTimestamp uint32
}

// BuildRtcpCname monta o CNAME de 18 bytes a partir de 12 bytes de entropia.
func BuildRtcpCname(entropy [12]byte) [RtcpCnameLen]byte {
	const hexChars = "0123456789abcdef"
	var h [11]byte
	for n := range h {
		b := entropy[6+n/2]
		if n&1 == 0 {
			h[n] = hexChars[b>>4]
		} else {
			h[n] = hexChars[b&0x0f]
		}
	}
	var c [RtcpCnameLen]byte
	copy(c[:5], h[:5])
	copy(c[5:8], "@pj")
	copy(c[8:14], h[5:])
	copy(c[14:], ".org")
	return c
}

// BuildSenderReport monta o Sender Report de 28 bytes (PT 200, sem blocos de recepção). nowMs é o
// relógio de parede em ms.
func BuildSenderReport(ssrc uint32, st RtcpSenderStats, nowMs uint64) [28]byte {
	var b [28]byte
	b[0] = 0x80
	b[1] = rtcpPtSr
	b[3] = 6
	binary.BigEndian.PutUint32(b[4:8], ssrc)
	binary.BigEndian.PutUint32(b[8:12], uint32(nowMs/1000+ntpUnixOffsetSecs))
	binary.BigEndian.PutUint32(b[12:16], uint32(float64(nowMs%1000)/1000.0*4294967296.0))
	binary.BigEndian.PutUint32(b[16:20], st.RtpTimestamp)
	binary.BigEndian.PutUint32(b[20:24], st.PacketsSent)
	binary.BigEndian.PutUint32(b[24:28], st.OctetsSent)
	return b
}

// BuildSourceDescription monta o SDES de um chunk (32 bytes). profileExtension liga o bit 4 do primeiro
// byte, que o vídeo do WhatsApp usa.
func BuildSourceDescription(ssrc uint32, cname *[RtcpCnameLen]byte, profileExtension bool) [32]byte {
	var p [32]byte
	p[0] = 0x81
	if profileExtension {
		p[0] |= 0x10
	}
	p[1] = rtcpPtSdes
	binary.BigEndian.PutUint16(p[2:4], 7)
	binary.BigEndian.PutUint32(p[4:8], ssrc)
	p[8] = 1
	p[9] = RtcpCnameLen
	copy(p[10:28], cname[:])
	return p
}

// BuildSenderReportWithSdes monta o pacote composto periódico SR+SDES (60 bytes), o "74 bytes protegido"
// do design (60 + 14 do SRTCP).
func BuildSenderReportWithSdes(ssrc uint32, st RtcpSenderStats, nowMs uint64, cname *[RtcpCnameLen]byte, profileExtension bool) []byte {
	sr := BuildSenderReport(ssrc, st, nowMs)
	if profileExtension {
		sr[0] |= 0x10
	}
	sdes := BuildSourceDescription(ssrc, cname, profileExtension)
	return append(sr[:], sdes[:]...)
}

// BuildPictureLossIndication monta o PLI (PT 206, fmt 1) de 12 bytes: pede ao contato um quadro-chave.
func BuildPictureLossIndication(senderSsrc, mediaSsrc uint32, profileExtension bool) [12]byte {
	var p [12]byte
	p[0] = 0x81
	if profileExtension {
		p[0] |= 0x10
	}
	p[1] = rtcpPtPsfb
	binary.BigEndian.PutUint16(p[2:4], 2)
	binary.BigEndian.PutUint32(p[4:8], senderSsrc)
	binary.BigEndian.PutUint32(p[8:12], mediaSsrc)
	return p
}

// RtcpRequestsKeyframe diz se o RTCP composto traz PLI (fmt 1) ou FIR (fmt 4) para o nosso SSRC de vídeo.
func RtcpRequestsKeyframe(data []byte, localVideoSsrc uint32) bool {
	for off := 0; off+4 <= len(data); {
		if (data[off]>>6)&0x03 != 2 {
			return false
		}
		n := (int(binary.BigEndian.Uint16(data[off+2:off+4])) + 1) * 4
		if n < 8 || off+n > len(data) {
			return false
		}
		p := data[off : off+n]
		if p[1] == rtcpPtPsfb && len(p) >= 12 {
			f := p[0] & 0x1f
			if f&0x10 != 0 {
				f &= 0x0f
			}
			media := binary.BigEndian.Uint32(p[8:12])
			switch f {
			case 1:
				if media == localVideoSsrc {
					return true
				}
			case 4:
				if media == localVideoSsrc {
					return true
				}
				for fci := p[12:]; len(fci) >= 8; fci = fci[8:] {
					if binary.BigEndian.Uint32(fci[:4]) == localVideoSsrc {
						return true
					}
				}
			}
		}
		off += n
	}
	return false
}

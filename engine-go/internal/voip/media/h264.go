package media

// H.264 RTP packetization/depacketization (RFC 6184) for WhatsApp video calls.
//
// Portado de purpshell/meowcaller (rtp/h264.go, MIT, Rajeh Taher), que por sua vez o traz do WaCalls
// feat/video-calls (jotadev66, MIT). Procedência e alterações em ../NOTICE.md.

const (
	h264StapAType = 24 // STAP-A aggregation packet NAL type (RFC 6184 §5.7.1)
	h264FuaType   = 28 // FU-A fragmentation unit NAL type (RFC 6184 §5.8)
	mtuPayloadMax = 800

	maxFuReassemblyBytes = 4 << 20
	maxAccessUnitBytes   = 8 << 20
)

// NAL types (H.264 spec / RFC 6184).
const (
	nalTypeSEI = 6
	nalTypeAUD = 9
)

// fuaFragSize é o tamanho do corpo de cada fragmento FU-A (MTU - indicator - FU header).
const fuaFragSize = mtuPayloadMax - 2

// PacketizeAu packetiza uma access unit Annex-B em payloads RTP conforme RFC 6184 e o que o
// WhatsApp espera: NALs não-VCL pequenos (SPS, PPS) são agregados em STAP-A; cada NAL VCL
// (slice) vai como single-NAL (se couber) ou FU-A (se não couber); AUD é omitido; SEI antes
// do primeiro VCL é omitido. Source of truth: whatsapp-rust wacore/src/voip/h264.rs packetize_au.
func PacketizeAu(au []byte) [][]byte {
	var out [][]byte
	seenVCL := false
	var stapBuf []byte
	var stapF, stapNri byte
	stapOpen := false

	flushStap := func() {
		if !stapOpen {
			return
		}
		stapBuf[0] = stapF | stapNri | byte(h264StapAType)
		out = append(out, stapBuf)
		stapBuf = nil
		stapOpen = false
	}

	for _, nal := range SplitAnnexB(au) {
		if len(nal) == 0 {
			continue
		}
		unitType := nal[0] & 0x1f
		if unitType == nalTypeAUD {
			continue
		}
		if !seenVCL && unitType == nalTypeSEI {
			continue
		}
		if unitType >= 1 && unitType <= 5 {
			flushStap()
			seenVCL = true
			if len(nal) <= mtuPayloadMax {
				p := make([]byte, len(nal))
				copy(p, nal)
				out = append(out, p)
			} else {
				out = append(out, fragmentNAL(nal)...)
			}
			continue
		}
		stapCost := 2 + len(nal)
		fits := false
		if stapOpen {
			fits = len(stapBuf)+stapCost <= mtuPayloadMax
		} else {
			fits = stapCost < mtuPayloadMax
		}
		if !fits {
			flushStap()
			if stapCost >= mtuPayloadMax {
				if len(nal) <= mtuPayloadMax {
					p := make([]byte, len(nal))
					copy(p, nal)
					out = append(out, p)
				} else {
					out = append(out, fragmentNAL(nal)...)
				}
				continue
			}
		}
		if !stapOpen {
			stapBuf = []byte{0}
			stapF = nal[0] & 0x80
			stapNri = nal[0] & 0x60
			stapOpen = true
		} else {
			stapF |= nal[0] & 0x80
			if nri := nal[0] & 0x60; nri > stapNri {
				stapNri = nri
			}
		}
		stapBuf = append(stapBuf, byte(len(nal)>>8), byte(len(nal)))
		stapBuf = append(stapBuf, nal...)
	}
	flushStap()
	return out
}

// fragmentNAL divide um NAL grande em fragmentos FU-A (RFC 6184 §5.8).
func fragmentNAL(nal []byte) [][]byte {
	indicator := (nal[0] & 0xe0) | byte(h264FuaType)
	origType := nal[0] & 0x1f
	body := nal[1:]
	nFrags := (len(body) + fuaFragSize - 1) / fuaFragSize
	var out [][]byte
	for i, chunk := 0, body; len(chunk) > 0; i++ {
		end := fuaFragSize
		if end > len(chunk) {
			end = len(chunk)
		}
		frag := chunk[:end]
		fuHeader := origType
		if i == 0 {
			fuHeader |= 0x80 // Start
		}
		if i == nFrags-1 {
			fuHeader |= 0x40 // End
		}
		pkt := make([]byte, 2+len(frag))
		pkt[0] = indicator
		pkt[1] = fuHeader
		copy(pkt[2:], frag)
		out = append(out, pkt)
		chunk = chunk[end:]
	}
	return out
}

// AUHasIDR reports whether an Annex-B access unit contains an IDR NAL unit.
func AUHasIDR(au []byte) bool {
	for _, nalu := range SplitAnnexB(au) {
		if len(nalu) > 0 && nalu[0]&0x1f == 5 {
			return true
		}
	}
	return false
}

// PackAccessUnit monta o payload de uma access unit como o WhatsApp a espera: ela inteira vira UM NAL,
// com os NALs separados por 00 00 00 01, sem start code no início e sem o AUD (tipo 9). Devolve nil se
// não sobrar nenhum NAL.
func PackAccessUnit(annexB []byte) []byte {
	var packed []byte
	for _, n := range SplitAnnexB(annexB) {
		if len(n) == 0 || n[0]&0x1f == 9 {
			continue
		}
		if len(packed) > 0 {
			packed = append(packed, 0, 0, 0, 1)
		}
		packed = append(packed, n...)
	}
	return packed
}

// PackageH264NALU splits one NAL unit into RTP payloads: a single payload when it fits
// the MTU budget, else FU-A fragments.
func PackageH264NALU(nalu []byte) [][]byte {
	if len(nalu) == 0 {
		return nil
	}

	if len(nalu) <= mtuPayloadMax {
		out := make([]byte, len(nalu))
		copy(out, nalu)
		return [][]byte{out}
	}
	naluHeader := nalu[0]
	fbitAndNri := naluHeader & 0xE0
	originalType := naluHeader & 0x1F
	fuIndicator := fbitAndNri | h264FuaType

	body := nalu[1:]
	fragSize := mtuPayloadMax - 2

	var out [][]byte
	offset := 0
	for offset < len(body) {
		end := offset + fragSize
		if end > len(body) {
			end = len(body)
		}
		chunk := body[offset:end]

		fuHeader := originalType
		if offset == 0 {
			fuHeader |= 0x80
		}
		if end == len(body) {
			fuHeader |= 0x40
		}

		pkt := make([]byte, 2+len(chunk))
		pkt[0] = fuIndicator
		pkt[1] = fuHeader
		copy(pkt[2:], chunk)
		out = append(out, pkt)

		offset = end
	}
	return out
}

// PackageH264STAPA aggregates several small NAL units into one STAP-A payload, or nil
// when there are fewer than two or the aggregate would exceed the MTU budget.
func PackageH264STAPA(nalus [][]byte) []byte {
	if len(nalus) < 2 {
		return nil
	}
	total := 1
	var maxNri byte
	for _, n := range nalus {
		if len(n) == 0 || len(n) > 0xFFFF {
			return nil
		}
		total += 2 + len(n)
		if total > mtuPayloadMax {
			return nil
		}
		nri := n[0] & 0x60
		if nri > maxNri {
			maxNri = nri
		}
	}

	out := make([]byte, total)
	out[0] = maxNri | h264StapAType
	offset := 1
	for _, n := range nalus {
		out[offset] = byte(len(n) >> 8)
		out[offset+1] = byte(len(n))
		offset += 2
		copy(out[offset:], n)
		offset += len(n)
	}
	return out
}

// SplitAnnexB splits an Annex-B byte stream (00 00 01 / 00 00 00 01 start codes) into
// its constituent NAL units, trimming trailing zero bytes before each start code.
func SplitAnnexB(data []byte) [][]byte {
	var nalus [][]byte
	start := -1
	i := 0
	for i < len(data) {
		sc := annexBStartCodeLen(data, i)
		if sc > 0 {
			if start >= 0 {
				end := i
				for end > start && data[end-1] == 0 {
					end--
				}
				if end > start {
					nalus = append(nalus, data[start:end])
				}
			}
			i += sc
			start = i
			continue
		}
		i++
	}
	if start >= 0 && start < len(data) {
		nalus = append(nalus, data[start:])
	}
	return nalus
}

// annexBStartCodeLen returns the length (3 or 4) of an Annex-B start code at offset, or 0.
func annexBStartCodeLen(data []byte, offset int) int {
	if offset+3 < len(data) &&
		data[offset] == 0 && data[offset+1] == 0 &&
		data[offset+2] == 0 && data[offset+3] == 1 {
		return 4
	}
	if offset+2 < len(data) &&
		data[offset] == 0 && data[offset+1] == 0 && data[offset+2] == 1 {
		return 3
	}
	return 0
}

// H264Depacketizer reassembles H.264 NAL units from RFC 6184 RTP payloads (single NAL,
// STAP-A, FU-A), carrying FU-A fragment state across packets.
type H264Depacketizer struct {
	fuBuf    []byte
	fuType   byte
	fuNriF   byte
	fuActive bool
	overflow bool
}

// H264AccessUnitAssembler reconstructs complete Annex-B access units while
// withholding damaged interframes until an IDR restores decoder state.
type H264AccessUnitAssembler struct {
	depacketizer   H264Depacketizer
	accessUnit     []byte
	expectedSeq    uint16
	hasSequence    bool
	keyframeNeeded bool
}

// Push consumes one ordered RTP payload and returns a complete, decodable access
// unit when the marker closes the frame.
func (a *H264AccessUnitAssembler) Push(sequence uint16, marker bool, payload []byte) ([]byte, bool, bool) {
	recoveryNeeded := false
	if a.hasSequence && sequence != a.expectedSeq {
		recoveryNeeded = !a.keyframeNeeded
		a.depacketizer = H264Depacketizer{}
		a.accessUnit = nil
		a.keyframeNeeded = true
	}
	a.hasSequence = true
	a.expectedSeq = sequence + 1

	for _, nalu := range a.depacketizer.Depacketize(payload) {
		if len(a.accessUnit)+4+len(nalu) > maxAccessUnitBytes {
			recoveryNeeded = recoveryNeeded || !a.keyframeNeeded
			a.depacketizer = H264Depacketizer{}
			a.accessUnit = nil
			a.keyframeNeeded = true
			break
		}
		a.accessUnit = append(a.accessUnit, 0x00, 0x00, 0x00, 0x01)
		a.accessUnit = append(a.accessUnit, nalu...)
	}
	if a.depacketizer.overflow {
		recoveryNeeded = recoveryNeeded || !a.keyframeNeeded
		a.depacketizer = H264Depacketizer{}
		a.accessUnit = nil
		a.keyframeNeeded = true
	}
	if !marker {
		return nil, false, recoveryNeeded
	}

	accessUnit := a.accessUnit
	a.accessUnit = nil
	a.depacketizer = H264Depacketizer{}
	if len(accessUnit) == 0 {
		return nil, false, recoveryNeeded
	}
	if a.keyframeNeeded {
		if !AUHasIDR(accessUnit) {
			return nil, false, recoveryNeeded
		}
		a.keyframeNeeded = false
	}
	return accessUnit, true, recoveryNeeded
}

// Depacketize consumes one RTP payload and returns the NAL units it produced (zero for a
// mid-fragment, one for single/FU-A completion, many for STAP-A).
func (d *H264Depacketizer) Depacketize(payload []byte) [][]byte {
	if len(payload) < 1 {
		return nil
	}
	naluType := payload[0] & 0x1F
	fbitAndNri := payload[0] & 0xE0

	switch {
	case naluType >= 1 && naluType <= 23:
		d.fuActive = false
		out := make([]byte, len(payload))
		copy(out, payload)
		return [][]byte{out}

	case naluType == h264StapAType:
		d.fuActive = false
		body := payload[1:]
		var out [][]byte
		for len(body) >= 2 {
			size := int(body[0])<<8 | int(body[1])
			body = body[2:]
			if size <= 0 || size > len(body) {
				return out
			}
			nalu := make([]byte, size)
			copy(nalu, body[:size])
			out = append(out, nalu)
			body = body[size:]
		}
		return out

	case naluType == h264FuaType:
		if len(payload) < 2 {
			d.fuActive = false
			return nil
		}
		fuHeader := payload[1]
		startBit := fuHeader & 0x80
		endBit := fuHeader & 0x40
		origType := fuHeader & 0x1F
		body := payload[2:]

		if startBit != 0 {
			if 1+len(body) > maxFuReassemblyBytes {
				d.fuActive = false
				d.fuBuf = d.fuBuf[:0]
				d.overflow = true
				return nil
			}
			d.fuActive = true
			d.fuType = origType
			d.fuNriF = fbitAndNri
			d.fuBuf = append(d.fuBuf[:0], fbitAndNri|origType)
			d.fuBuf = append(d.fuBuf, body...)
		} else if d.fuActive {
			if len(d.fuBuf)+len(body) > maxFuReassemblyBytes {
				d.fuActive = false
				d.fuBuf = d.fuBuf[:0]
				d.overflow = true
				return nil
			}
			d.fuBuf = append(d.fuBuf, body...)
		} else {
			return nil
		}

		if endBit != 0 && d.fuActive {
			d.fuActive = false
			out := make([]byte, len(d.fuBuf))
			copy(out, d.fuBuf)
			d.fuBuf = d.fuBuf[:0]
			return [][]byte{out}
		}
		return nil

	default:
		d.fuActive = false
		return nil
	}
}

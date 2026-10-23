package mlow

// MLow's multiframe envelope: several complete MLow packets carried in one RTP payload.
//
// This is how the real client reaches a packet longer than its frame length. Its encoder caches a
// fixed three blocks of 20 ms, so a frame_length_ms of 120 does not produce one 120 ms block: it
// produces two 60 ms blocks aggregated here. A decoder that does not implement this reads the
// envelope's first byte as a TOC, finds bit 7 set, calls it a SID, and emits comfort noise for
// every packet of the call. The call goes completely silent with no error anywhere.
//
// The envelope indicator is (b & 0xC0) != 0xC0 && (b & 0x82) == 0x82. That point is unreachable
// as a real TOC by construction: bit 7 is SID and bit 1 is FEC, and the encoder asserts a SID frame
// never carries FEC, so no frame it emits can have both set. The collision is deliberate.
//
// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/main/wacore/src/voip/mlow/multiframe.rs

const maxEnvelopeFrames = 18
const countMask = 0x3f
const paddingFlag = 0x40

type MultiframeError struct{ Msg string }

func (e *MultiframeError) Error() string { return e.Msg }

var (
	ErrEnvelopeTruncated     = &MultiframeError{"the envelope ended before its header did"}
	ErrEnvelopeBadFrameCount = &MultiframeError{"frame count is outside 1..=18"}
	ErrEnvelopeLengthOverrun = &MultiframeError{"a declared frame length runs past the end of the envelope"}
	ErrEnvelopeEmptySubFrame = &MultiframeError{"a sub-frame is empty"}
)

// IsMultiframe reports whether the payload is a multiframe envelope rather than a single MLow frame.
//
// Checked before the TOC is interpreted, because under the TOC grammar these bytes read as a SID
// and would be silently turned into comfort noise.
func IsMultiframe(payload []byte) bool {
	return len(payload) >= 2 && payload[0]&0xC0 != 0xC0 && payload[0]&0x82 == 0x82
}

// SplitMultiframe splits one envelope into the complete MLow packets it carries, in transmission order.
//
// Each element is a whole packet with its own TOC, decodable exactly as if it had arrived alone.
// The caller decodes them in order and concatenates: the envelope's RTP timestamp belongs to the
// LAST sub-frame, and the earlier ones run backwards from it, so playing them in array order
// is what puts the audio back in time order.
//
// Borrows the input; no allocation beyond the returned slice of slices, which is bounded by maxEnvelopeFrames.
func SplitMultiframe(payload []byte) ([][]byte, error) {
	if len(payload) < 2 {
		return nil, ErrEnvelopeTruncated
	}
	countByte := payload[1]
	frames := int(countByte & countMask)
	if frames == 0 || frames > maxEnvelopeFrames {
		return nil, ErrEnvelopeBadFrameCount
	}
	rest := payload[2:]

	// Opus padding: a run of 255s, then a final byte giving the remaining padding length.
	padding := 0
	if countByte&paddingFlag != 0 {
		for {
			if len(rest) == 0 {
				return nil, ErrEnvelopeTruncated
			}
			b := rest[0]
			rest = rest[1:]
			if b == 255 {
				padding += 254
			} else {
				padding += int(b)
				break
			}
		}
	}

	// One explicit length per frame except the last, which takes whatever is left.
	// Same one-or-two byte encoding standard Opus uses.
	lengths := make([]int, frames)
	declared := 0
	for i := 0; i < frames-1; i++ {
		if len(rest) == 0 {
			return nil, ErrEnvelopeTruncated
		}
		first := rest[0]
		var length int
		if first < 252 {
			length = int(first)
			rest = rest[1:]
		} else {
			if len(rest) < 2 {
				return nil, ErrEnvelopeTruncated
			}
			length = int(first) + int(rest[1])*4
			rest = rest[2:]
		}
		lengths[i] = length
		declared += length
	}

	bodies := len(rest) - padding
	if bodies < declared {
		return nil, ErrEnvelopeLengthOverrun
	}
	last := bodies - declared
	if last == 0 {
		return nil, ErrEnvelopeEmptySubFrame
	}
	lengths[frames-1] = last

	out := make([][]byte, 0, frames)
	cursor := rest
	for _, length := range lengths {
		if length == 0 {
			return nil, ErrEnvelopeEmptySubFrame
		}
		if length > len(cursor) {
			return nil, ErrEnvelopeLengthOverrun
		}
		out = append(out, cursor[:length:length])
		cursor = cursor[length:]
	}
	return out, nil
}

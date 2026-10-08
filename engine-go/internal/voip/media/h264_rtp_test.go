package media

import (
	"encoding/hex"
	"testing"
)

func mustHexB(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func encodeHdr(t *testing.T, h *RtpHeader) []byte {
	t.Helper()
	buf := make([]byte, h.Size())
	if _, err := h.Encode(buf); err != nil {
		t.Fatal(err)
	}
	return buf
}

// Vetor de uma captura real de um Android (meowcaller rtp/testdata/kats.json). Se os bytes mudarem, o
// WhatsApp deixa de tratar o quadro como keyframe.
func TestVideoHeaderMatchesAndroidCapture(t *testing.T) {
	h := NewRtpHeader(PayloadTypeH264, 1, 114120, 0x49c5fb8c)
	h.Marker = true
	h.Extension = true
	h.ExtensionProfile = WhatsappRtpExtensionProfile
	h.ExtensionData = (&VideoRtpExtension{MediaFrameInfo: 0x09, ShortOffset: 29, TransportSequence: 0x0c3f}).Encode()

	const want = "90e100010001bdc849c5fb8cdebe0003300951000061001d910c3f00"
	if got := hex.EncodeToString(encodeHdr(t, h)); got != want {
		t.Fatalf("cabeçalho de vídeo\n got  %s\n want %s", got, want)
	}
	if h.Size() != 28 {
		t.Fatalf("tamanho do cabeçalho = %d, esperado 28", h.Size())
	}
}

// Extensões de uma captura real de um navegador web: 1º pacote de uma AU leva FrameNumber, os demais não.
func TestVideoStreamMatchesCapturedWebFrameMetadata(t *testing.T) {
	s := NewVideoRtpStream(0x11223344, 4500)
	_, e1 := s.NextPacket(false, VideoFrameInfoIDR)
	_, e2 := s.NextPacket(true, VideoFrameInfoIDR)
	_, e3 := s.NextPacket(true, VideoFrameInfoDelta)
	for i, c := range []struct {
		got  *VideoRtpExtension
		want string
	}{
		{e1, "32080001510000610000910000000000"},
		{e2, "300851000061000091000100"},
		{e3, "32200002510000610000910002000000"},
	} {
		if got := hex.EncodeToString(c.got.Encode()); got != c.want {
			t.Errorf("pacote %d: extensão %s, esperado %s", i+1, got, c.want)
		}
	}
}

func TestVideoStreamUsesOneTimestampPerAccessUnit(t *testing.T) {
	s := NewVideoRtpStream(0x11223344, 4500)
	h1, x1 := s.NextPacket(false, VideoFrameInfoIDR)
	h2, x2 := s.NextPacket(true, VideoFrameInfoIDR)
	h3, x3 := s.NextPacket(true, VideoFrameInfoDelta)
	if h1.Timestamp != 0 || h2.Timestamp != 0 || h3.Timestamp != 4500 {
		t.Errorf("timestamps = %d,%d,%d, esperado 0,0,4500", h1.Timestamp, h2.Timestamp, h3.Timestamp)
	}
	if h1.SequenceNumber != 1 || h2.SequenceNumber != 2 || h3.SequenceNumber != 3 {
		t.Errorf("sequências = %d,%d,%d", h1.SequenceNumber, h2.SequenceNumber, h3.SequenceNumber)
	}
	if h1.Marker || !h2.Marker || !h3.Marker {
		t.Errorf("marker = %v,%v,%v, esperado false,true,true", h1.Marker, h2.Marker, h3.Marker)
	}
	if x1.TransportSequence != 0 || x2.TransportSequence != 1 || x3.TransportSequence != 2 {
		t.Errorf("sequência de transporte = %d,%d,%d", x1.TransportSequence, x2.TransportSequence, x3.TransportSequence)
	}
}

func TestParseVideoExtensionRoundTripAndForeignProfile(t *testing.T) {
	s := NewVideoRtpStream(1, 3000)
	h, want := s.NextPacket(false, VideoFrameInfoIDR)
	dec, err := DecodeRtpHeader(encodeHdr(t, h))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := ParseVideoRtpExtension(dec)
	if !ok || got.MediaFrameInfo != want.MediaFrameInfo || got.FrameNumber == nil || *got.FrameNumber != 1 {
		t.Fatalf("round-trip falhou: %+v ok=%v", got, ok)
	}
	dec.ExtensionProfile = 0x1234
	if _, ok := ParseVideoRtpExtension(dec); ok {
		t.Fatal("perfil de extensão estranho não pode ser lido como vídeo")
	}
	if _, ok := ParseVideoRtpExtension(nil); ok {
		t.Fatal("cabeçalho nulo")
	}
}

// Captura real de orientação (iPhone/Android em retrato): o valor 3 vem dos 2 bits baixos do MediaFrameInfo.
func TestParseCapturedVideoOrientation(t *testing.T) {
	pkt := mustHexB(t, "906100010003e77e0ba3152bdebe0002300b510000610002")
	h, err := DecodeRtpHeader(pkt)
	if err != nil {
		t.Fatal(err)
	}
	ext, ok := ParseVideoRtpExtension(h)
	if !ok {
		t.Fatal("captura real rejeitada")
	}
	if got := ext.DisplayOrientation(); got != 3 {
		t.Fatalf("orientação = %d, esperado 3", got)
	}
	for fi, want := range map[uint8]int{0x20: 0, 0x21: 1, 0x22: 2, 0x23: 3, 0x33: 3, 0x0b: 3} {
		if got := (&VideoRtpExtension{MediaFrameInfo: fi}).DisplayOrientation(); got != want {
			t.Errorf("frameInfo %#x → %d, esperado %d", fi, got, want)
		}
	}
}

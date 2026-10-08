package media

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Vetor da meowcaller (rtcp.senderReport do kats.json).
func TestSenderReportMatchesReferenceVector(t *testing.T) {
	sr := BuildSenderReport(0x12345678, RtcpSenderStats{PacketsSent: 5, OctetsSent: 600, RtpTimestamp: 1600}, 1718000000000)
	if got, want := hex.EncodeToString(sr[:]), "80c8000612345678ea11180000000000000006400000000500000258"; got != want {
		t.Fatalf("SR = %s, want %s", got, want)
	}
}

func TestSenderReportWithSdesVideoProfile(t *testing.T) {
	cname := BuildRtcpCname([12]byte{0, 1, 2, 3, 4, 5, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb})
	if string(cname[:]) != "66778@pj899aab.org" {
		t.Fatalf("cname = %q", cname)
	}
	c := BuildSenderReportWithSdes(0x11112222, RtcpSenderStats{PacketsSent: 3, OctetsSent: 400, RtpTimestamp: 90000}, 1700000000000, &cname, true)
	if len(c) != 60 {
		t.Fatalf("composto = %d bytes, quer 60", len(c))
	}
	if c[0] != 0x90 || c[1] != rtcpPtSr || c[28] != 0x91 || c[29] != rtcpPtSdes {
		t.Fatalf("cabeçalhos = %x / %x", c[:2], c[28:30])
	}
	k := srtcpKeysFor(t, bytes.Repeat([]byte{9}, 32), "1:0@lid")
	prot, err := k.Protect(0x11112222, 1, c)
	if err != nil {
		t.Fatal(err)
	}
	if len(prot) != 74 {
		t.Fatalf("protegido = %d bytes, o design manda 74", len(prot))
	}
}

func TestPictureLossIndicationRoundTrip(t *testing.T) {
	got := BuildPictureLossIndication(0x11112222, 0x55556666, true)
	want := [12]byte{0x91, rtcpPtPsfb, 0, 2, 0x11, 0x11, 0x22, 0x22, 0x55, 0x55, 0x66, 0x66}
	if got != want {
		t.Fatalf("PLI = %x, want %x", got, want)
	}
	if !RtcpRequestsKeyframe(got[:], 0x55556666) {
		t.Fatal("o PLI montado não pede o próprio SSRC")
	}
	if RtcpRequestsKeyframe(got[:], 0x99990000) {
		t.Fatal("PLI de outro SSRC foi aceito")
	}
}

func TestRtcpRequestsKeyframeFIRAndCompound(t *testing.T) {
	const local = 0x55556666
	fir := []byte{0x84, rtcpPtPsfb, 0, 4, 1, 1, 1, 1, 0, 0, 0, 0, 0x55, 0x55, 0x66, 0x66, 7, 0, 0, 0}
	if !RtcpRequestsKeyframe(fir, local) {
		t.Fatal("FIR para o SSRC local não foi reconhecido")
	}
	sr := BuildSenderReport(1, RtcpSenderStats{}, 0)
	pli := BuildPictureLossIndication(2, local, false)
	if !RtcpRequestsKeyframe(append(sr[:], pli[:]...), local) {
		t.Fatal("PLI depois de um SR no mesmo pacote composto não foi reconhecido")
	}
	if RtcpRequestsKeyframe(sr[:], local) {
		t.Fatal("um SR sozinho não pede quadro-chave")
	}
	if RtcpRequestsKeyframe([]byte{0x80, rtcpPtPsfb, 0xff, 0xff, 0, 0, 0, 0}, local) {
		t.Fatal("comprimento inválido não pode ser lido além do pacote")
	}
}

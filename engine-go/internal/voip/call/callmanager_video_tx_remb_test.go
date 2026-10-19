package call

import (
	"testing"
)

func TestPeerREMB_UpdatesEstimatedBitrate(t *testing.T) {
	r := newTxRig(t)
	remb := []byte{
		0x8f, 206, 0, 5,
		0x11, 0x22, 0x33, 0x44,
		0, 0, 0, 0,
		'R', 'E', 'M', 'B',
		1,
		(10 << 2),
		1,
		68,
		0, 0, 0, 0,
	}
	wire, err := r.peerSk.Protect(0x11223344, 1, remb)
	if err != nil {
		t.Fatal(err)
	}
	r.m.onRelayData(wire)
	if got := r.m.PeerEstimatedBitrate(); got != 331776 {
		t.Fatalf("bitrate estimado = %d, esperado 331776", got)
	}
}

package media

import "testing"

func TestParseRembBitrateDirect(t *testing.T) {
	pkt := []byte{
		0x8f, 206, 0, 5,
		0, 0, 0, 0,
		0, 0, 0, 0,
		'R', 'E', 'M', 'B',
		1,
		(10 << 2),
		1,
		68,
		0, 0, 0, 0,
	}
	got := ParseRembBitrate(pkt)
	if got != 331776 {
		t.Fatalf("bitrate = %d, esperado 331776", got)
	}
}

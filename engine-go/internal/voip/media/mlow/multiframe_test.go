package mlow

import (
	"testing"
)

func TestIsMultiframeIndicator(t *testing.T) {
	cases := []struct {
		bytes []byte
		want  bool
	}{
		{[]byte{0x92, 0x02}, true},  // bit 7+1 set, bit 6 clear
		{[]byte{0x82, 0x02}, true},  // bit 7+1 set, bit 6 clear
		{[]byte{0x92}, false},       // too short
		{[]byte{0x50, 0xAA}, false}, // bit 7 clear
		{[]byte{0xD2, 0x02}, false}, // bit 6+7 set = standard Opus
		{[]byte{0x90, 0x02}, false}, // bit 7 set but bit 1 clear
	}
	for _, c := range cases {
		if got := IsMultiframe(c.bytes); got != c.want {
			t.Errorf("IsMultiframe(%v) = %v, want %v", c.bytes, got, c.want)
		}
	}
}

func TestSplitMultiframeTwoFrames(t *testing.T) {
	// Two 60 ms blocks in one envelope (what the real client emits for 120 ms frame length)
	first := []byte{0x50, 0xAA, 0xBB, 0xCC}
	second := []byte{0x50, 0xDD, 0xEE, 0xFF}
	envelope := []byte{0x82 | (first[0] & 0x39), 0x02, byte(len(first))}
	envelope = append(envelope, first...)
	envelope = append(envelope, second...)

	frames, err := SplitMultiframe(envelope)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	if string(frames[0]) != string(first) {
		t.Errorf("frame 0: got %v, want %v", frames[0], first)
	}
	if string(frames[1]) != string(second) {
		t.Errorf("frame 1: got %v, want %v", frames[1], second)
	}
}

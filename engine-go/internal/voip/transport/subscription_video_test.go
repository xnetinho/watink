package transport

import (
	"bytes"
	"testing"
)

// Sem o SSRC de vídeo do contato na alocação, o relay não encaminha o vídeo dele.
func TestAllocationListIncludesVideoSsrcs(t *testing.T) {
	m := NewSctpRelayManager(nil)
	m.SetSsrc(0xA0)
	m.SetSubscriptionSsrc(0xB0)
	m.SetExtraSsrcs([]uint32{0xA2}, []uint32{0xB2})

	m.mu.Lock()
	got := m.allocationSsrcList()
	m.mu.Unlock()
	want := BuildSSRCSubscriptionList([]uint32{0xA0, 0xA2}, []uint32{0xB0, 0xB2}, 0, 0)
	if !bytes.Equal(got, want) {
		t.Fatalf("lista de alocação\n got  %x\n want %x", got, want)
	}
}

// Chamada só de voz: a lista é exatamente a de antes (áudio nosso + áudio do contato).
func TestAllocationListWithoutVideoIsAudioOnly(t *testing.T) {
	m := NewSctpRelayManager(nil)
	m.SetSsrc(0xA0)
	m.SetSubscriptionSsrc(0xB0)
	m.mu.Lock()
	got := m.allocationSsrcList()
	m.mu.Unlock()
	if want := BuildSSRCSubscriptionList([]uint32{0xA0}, []uint32{0xB0}, 0, 0); !bytes.Equal(got, want) {
		t.Fatalf("regressão da voz: %x != %x", got, want)
	}
}

// Os SSRCs de vídeo de uma chamada não podem vazar para a próxima.
func TestCleanupDropsExtraSsrcs(t *testing.T) {
	m := NewSctpRelayManager(nil)
	m.SetSsrc(0xA0)
	m.SetExtraSsrcs([]uint32{0xA2}, []uint32{0xB2})
	m.Cleanup()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.extraSelfSsrcs) != 0 || len(m.extraPeerSsrcs) != 0 {
		t.Fatalf("SSRCs extras sobraram: %v %v", m.extraSelfSsrcs, m.extraPeerSsrcs)
	}
}

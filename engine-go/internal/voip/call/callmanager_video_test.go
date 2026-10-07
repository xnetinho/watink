package call

import (
	"context"
	"log/slog"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/media"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/transport"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

// relayEspiao registra o que o CallManager entrega ao relay, sem rede.
type relayEspiao struct {
	extraSelf, extraPeer []uint32
}

func (r *relayEspiao) SetSsrc(uint32)                          {}
func (r *relayEspiao) SetSubscriptionSsrc(uint32)              {}
func (r *relayEspiao) SetExtraSsrcs(self, peer []uint32)       { r.extraSelf, r.extraPeer = self, peer }
func (r *relayEspiao) SetOnConnected(func(string, int))        {}
func (r *relayEspiao) SetOnReceive(func([]byte))               {}
func (r *relayEspiao) ResendSubscriptions()                    {}
func (r *relayEspiao) ConfigureRelays([]transport.RelayConfig) {}
func (r *relayEspiao) Broadcast([]byte)                        {}
func (r *relayEspiao) HasConnection() bool                     { return false }
func (r *relayEspiao) ConnectedCount() int                     { return 0 }
func (r *relayEspiao) Cleanup()                                {}

// sockComNumero é o fakeSock com um número próprio, como num engine conectado.
type sockComNumero struct{ fakeSock }

func (sockComNumero) OwnPN() types.JID {
	return types.NewJID("5511988887777", types.DefaultUserServer)
}

func videoOffer(callID, from string, withVideo bool) *waBinary.Node {
	kids := []waBinary.Node{{Tag: "audio"}}
	if withVideo {
		kids = append(kids, waBinary.Node{Tag: "video"})
	}
	return &waBinary.Node{Tag: "call", Attrs: waBinary.Attrs{"from": from}, Content: []waBinary.Node{
		{Tag: "offer", Attrs: waBinary.Attrs{"call-id": callID, "call-creator": from}, Content: kids},
	}}
}

func newVideoRig(t *testing.T, withVideo bool) (*CallManager, *relayEspiao, types.JID) {
	t.Helper()
	peer := types.NewJID("5511999990001", types.DefaultUserServer)
	m := NewCallManager(&sockComNumero{}, slog.Default())
	spy := &relayEspiao{}
	m.relay = spy
	m.DeferPreaccept = true
	m.HandleCallOffer(context.Background(), videoOffer("CALLV", peer.String(), withVideo), peer)
	return m, spy, peer
}

// O relay só encaminha o vídeo do contato se o SSRC de vídeo dele estiver na alocação. O SSRC é o
// mesmo HKDF do áudio com o slot 2 (VideoSlotWord) no lugar do 0.
func TestIncomingVideoOffer_DerivesVideoSsrcsIntoRelay(t *testing.T) {
	m, spy, peer := newVideoRig(t, true)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentCall.MediaType != core.CallMediaTypeVideo {
		t.Fatalf("a oferta tem <video>, a chamada devia ser de vídeo: %v", m.currentCall.MediaType)
	}
	wantPeer := media.GenerateSecureSsrc("CALLV", peer.String(), media.VideoSlotWord)
	if m.videoPeerSsrc != wantPeer || m.videoPeerSsrc == 0 {
		t.Fatalf("SSRC de vídeo do contato = %#x, esperado %#x", m.videoPeerSsrc, wantPeer)
	}
	if m.videoPeerSsrc == m.peerSsrcs[0] {
		t.Fatal("o SSRC de vídeo não pode coincidir com o de áudio")
	}
	if len(spy.extraPeer) != 1 || spy.extraPeer[0] != wantPeer {
		t.Fatalf("o relay recebeu peer=%v, esperado [%#x]", spy.extraPeer, wantPeer)
	}
	if len(spy.extraSelf) != 1 || spy.extraSelf[0] != m.videoSelfSsrc || m.videoSelfSsrc == 0 {
		t.Fatalf("o relay recebeu self=%v, videoSelfSsrc=%#x", spy.extraSelf, m.videoSelfSsrc)
	}
}

// Chamada de voz: nenhum SSRC de vídeo, relay exatamente como antes.
func TestIncomingVoiceOffer_HasNoVideoSsrcs(t *testing.T) {
	m, spy, _ := newVideoRig(t, false)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.videoSelfSsrc != 0 || m.videoPeerSsrc != 0 {
		t.Fatalf("voz não deriva SSRC de vídeo: %#x %#x", m.videoSelfSsrc, m.videoPeerSsrc)
	}
	if len(spy.extraSelf) != 0 || len(spy.extraPeer) != 0 {
		t.Fatalf("o relay não pode receber extras na voz: %v %v", spy.extraSelf, spy.extraPeer)
	}
}

// Os SSRCs de uma chamada não podem sobrar para a seguinte.
func TestCleanupMediaClearsVideoSsrcs(t *testing.T) {
	m, _, _ := newVideoRig(t, true)
	m.cleanupMedia()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.videoSelfSsrc != 0 || m.videoPeerSsrc != 0 {
		t.Fatalf("SSRC de vídeo sobrou: %#x %#x", m.videoSelfSsrc, m.videoPeerSsrc)
	}
}

// Quando o JID do aparelho do contato é refinado (ack do relay), o SSRC de vídeo acompanha o de áudio.
func TestVideoSsrcFollowsRefinedPeerDevice(t *testing.T) {
	m, spy, _ := newVideoRig(t, true)
	m.mu.Lock()
	m.deriveVideoSsrcsLocked("", "5511999990001:7@s.whatsapp.net")
	m.applyVideoSsrcsLocked()
	want := media.GenerateSecureSsrc("CALLV", "5511999990001:7@s.whatsapp.net", media.VideoSlotWord)
	got := m.videoPeerSsrc
	m.mu.Unlock()
	if got != want || len(spy.extraPeer) != 1 || spy.extraPeer[0] != want {
		t.Fatalf("SSRC refinado = %#x, esperado %#x (relay=%v)", got, want, spy.extraPeer)
	}
}

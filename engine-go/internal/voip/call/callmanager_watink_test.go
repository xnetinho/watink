package call

import (
	"context"
	"log/slog"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

type fakeSock struct{ sent []waBinary.Node }

func (f *fakeSock) OwnPN() types.JID                                 { return types.EmptyJID }
func (f *fakeSock) OwnLID() types.JID                                { return types.EmptyJID }
func (f *fakeSock) AccountDeviceIdentityNode() (waBinary.Node, bool) { return waBinary.Node{}, false }
func (f *fakeSock) SendNode(_ context.Context, n waBinary.Node) error {
	f.sent = append(f.sent, n)
	return nil
}
func (f *fakeSock) Query(context.Context, waBinary.Node) (*waBinary.Node, error) { return nil, nil }
func (f *fakeSock) GetUSyncDevices(context.Context, []types.JID) ([]types.JID, error) {
	return nil, nil
}
func (f *fakeSock) AssertSessions(context.Context, []types.JID, bool) error { return nil }
func (f *fakeSock) CreateParticipantNodes(context.Context, []types.JID, []byte, waBinary.Attrs) ([]waBinary.Node, bool, error) {
	return nil, false, nil
}
func (f *fakeSock) DecryptCallKey(context.Context, types.JID, *waBinary.Node) ([]byte, error) {
	return nil, nil
}
func (f *fakeSock) GetTCToken(context.Context, types.JID) ([]byte, error)     { return nil, nil }
func (f *fakeSock) ResolveLIDForPN(_ context.Context, pn types.JID) types.JID { return pn }

func (f *fakeSock) tags() []string {
	var out []string
	for _, n := range f.sent {
		kids, _ := n.Content.([]waBinary.Node)
		for _, k := range kids {
			out = append(out, k.Tag)
		}
	}
	return out
}

func offerNode(callID, from string) *waBinary.Node {
	return &waBinary.Node{Tag: "call", Attrs: waBinary.Attrs{"from": from}, Content: []waBinary.Node{
		{Tag: "offer", Attrs: waBinary.Attrs{"call-id": callID, "call-creator": from}},
	}}
}

func TestDeferPreaccept_OfferSendsNothingUntilAsked(t *testing.T) {
	peer := types.NewJID("5511999990001", types.DefaultUserServer)
	sock := &fakeSock{}
	m := NewCallManager(sock, slog.Default())
	m.DeferPreaccept = true
	m.HandleCallOffer(context.Background(), offerNode("CID1", peer.String()), peer)

	if len(sock.sent) != 0 {
		t.Fatalf("nada pode ser enviado ao receber a oferta: %v", sock.tags())
	}
	if c := m.CurrentCall(); c == nil || !c.CanAccept() {
		t.Fatal("a chamada deve estar tocando")
	}
	if err := m.SendPreaccept(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := sock.tags(); len(got) != 1 || got[0] != "preaccept" {
		t.Fatalf("esperava só preaccept: %v", got)
	}
}

func TestWithoutDeferPreaccept_KeepsOriginalBehaviour(t *testing.T) {
	peer := types.NewJID("5511999990001", types.DefaultUserServer)
	sock := &fakeSock{}
	m := NewCallManager(sock, slog.Default())
	m.HandleCallOffer(context.Background(), offerNode("CID2", peer.String()), peer)
	if got := sock.tags(); len(got) != 1 || got[0] != "preaccept" {
		t.Fatalf("o padrão original manda preaccept na oferta: %v", got)
	}
}

func TestAbandonCall_SendsNothingAndFiresOnEnded(t *testing.T) {
	peer := types.NewJID("5511999990001", types.DefaultUserServer)
	sock := &fakeSock{}
	m := NewCallManager(sock, slog.Default())
	m.DeferPreaccept = true
	ended := 0
	m.OnEnded = func(c *CallInfo) {
		ended++
		if c.StateData.EndReason != core.EndCallReasonTimeout {
			t.Errorf("motivo=%s", c.StateData.EndReason)
		}
	}
	m.HandleCallOffer(context.Background(), offerNode("CID3", peer.String()), peer)
	m.AbandonCall(core.EndCallReasonTimeout)
	m.AbandonCall(core.EndCallReasonTimeout)

	if len(sock.sent) != 0 {
		t.Fatalf("abandonar não pode enviar reject nem terminate: %v", sock.tags())
	}
	if ended != 1 || !m.CurrentCall().IsEnded() {
		t.Fatalf("ended=%d", ended)
	}
	if err := m.SendPreaccept(context.Background()); err != nil || len(sock.sent) != 0 {
		t.Fatal("preaccept numa chamada encerrada deve ser no-op")
	}
}

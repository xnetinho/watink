package signaling

import (
	"bytes"
	"context"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

// Os blobs de capability vêm de capturas reais do WhatsApp (meowcaller) e o servidor valida a
// forma do offer. Este teste trava os bytes para uma troca acidental não passar despercebida.
var (
	wantOffer      = []byte{0x01, 0x05, 0xf7, 0x09, 0xe0, 0xbb, 0x13}
	wantVideoOffer = []byte{0x01, 0x05, 0xf7, 0x09, 0xe0, 0xfa, 0x13}
	wantPreaccept  = []byte{0x01, 0x05, 0xf7, 0x09, 0xe0, 0xbb, 0x07}
)

type capSock struct{}

func (capSock) OwnPN() types.JID  { return types.NewJID("5511999990000", types.DefaultUserServer) }
func (capSock) OwnLID() types.JID { return types.EmptyJID }
func (capSock) AccountDeviceIdentityNode() (waBinary.Node, bool) {
	return waBinary.Node{}, false
}
func (capSock) SendNode(context.Context, waBinary.Node) error { return nil }
func (capSock) Query(context.Context, waBinary.Node) (*waBinary.Node, error) {
	return nil, nil
}
func (capSock) GetUSyncDevices(_ context.Context, j []types.JID) ([]types.JID, error) { return j, nil }
func (capSock) AssertSessions(context.Context, []types.JID, bool) error               { return nil }
func (capSock) CreateParticipantNodes(context.Context, []types.JID, []byte, waBinary.Attrs) ([]waBinary.Node, bool, error) {
	return nil, false, nil
}
func (capSock) DecryptCallKey(context.Context, types.JID, *waBinary.Node) ([]byte, error) {
	return nil, nil
}
func (capSock) GetTCToken(context.Context, types.JID) ([]byte, error)     { return nil, nil }
func (capSock) ResolveLIDForPN(_ context.Context, pn types.JID) types.JID { return pn }

func capabilityOf(t *testing.T, n waBinary.Node, action string) []byte {
	t.Helper()
	kids, _ := n.Content.([]waBinary.Node)
	if len(kids) == 0 || kids[0].Tag != action {
		t.Fatalf("esperava <%s>, veio %+v", action, kids)
	}
	inner, _ := kids[0].Content.([]waBinary.Node)
	for _, c := range inner {
		if c.Tag == "capability" {
			b, _ := c.Content.([]byte)
			return b
		}
	}
	t.Fatalf("<%s> sem <capability>", action)
	return nil
}

func TestOfferCapability(t *testing.T) {
	peer := types.NewJID("5511999990001", types.DefaultUserServer)
	for _, tc := range []struct {
		name  string
		video bool
		want  []byte
	}{
		{"áudio", false, wantOffer},
		{"vídeo", true, wantVideoOffer},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, err := BuildOfferStanza(context.Background(), capSock{}, "CALL1", bytes.Repeat([]byte{1}, 32), peer, tc.video)
			if err != nil {
				t.Fatal(err)
			}
			if got := capabilityOf(t, n, "offer"); !bytes.Equal(got, tc.want) {
				t.Fatalf("capability do offer = %x, esperado %x", got, tc.want)
			}
		})
	}
}

func TestPreacceptCapability(t *testing.T) {
	peer := types.NewJID("5511999990001", types.DefaultUserServer)
	n := BuildPreacceptStanza(peer, "CALL1", peer, false)
	if got := capabilityOf(t, n, "preaccept"); !bytes.Equal(got, wantPreaccept) {
		t.Fatalf("capability do preaccept = %x, esperado %x", got, wantPreaccept)
	}
}

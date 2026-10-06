package call

import (
	"bytes"
	"context"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/media"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

type countingCodec struct{ decodes atomic.Int32 }

func (c *countingCodec) Encode([]float32) ([]byte, error) { return nil, nil }
func (c *countingCodec) Decode([]byte) ([]float32, error) {
	c.decodes.Add(1)
	return make([]float32, 960), nil
}
func (c *countingCodec) FrameSize() int  { return 960 }
func (c *countingCodec) SampleRate() int { return 16000 }
func (c *countingCodec) Close()          {}

const dedupPeerSsrc uint32 = 0x57414301

func dedupFixture(t *testing.T) (*CallManager, *countingCodec, func(seq uint16) []byte, *atomic.Int32) {
	t.Helper()
	km, err := media.DerivePerJidSrtpKey(bytes.Repeat([]byte{0x42}, 32), "peer:0@lid")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := media.NewSrtpSession(km, km, core.SRTPRecvAuthTagLen, core.SRTPRecvAuthTagLen)
	if err != nil {
		t.Fatal(err)
	}
	rx, err := media.NewSrtpSession(km, km, core.SRTPRecvAuthTagLen, core.SRTPRecvAuthTagLen)
	if err != nil {
		t.Fatal(err)
	}
	cod := &countingCodec{}
	m := NewCallManager(&fakeSock{}, slog.Default())
	m.srtpSession, m.codec = rx, cod
	m.selfSsrc = 0x11111111
	var delivered atomic.Int32
	m.OnPeerAudio = func([]float32) { delivered.Add(1) }
	wire := func(seq uint16) []byte {
		w, err := tx.Protect(&media.RtpPacket{
			Header:  media.NewRtpHeader(core.PayloadTypeWhatsAppOpus, seq, uint32(seq)*960, dedupPeerSsrc),
			Payload: bytes.Repeat([]byte{byte(seq)}, 40),
		})
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	return m, cod, wire, &delivered
}

func TestRelayCopiesAreDeliveredOnce(t *testing.T) {
	m, cod, wire, delivered := dedupFixture(t)
	for seq := uint16(1); seq <= 50; seq++ {
		w := wire(seq)
		m.onRelayData(w)
		m.onRelayData(w)
		m.onRelayData(w)
	}
	if got := cod.decodes.Load(); got != 50 {
		t.Fatalf("decodificações: %d, esperado 50 (3 relays entregando a mesma cópia)", got)
	}
	if got := delivered.Load(); got != 50 {
		t.Fatalf("quadros entregues ao operador: %d, esperado 50", got)
	}
}

func peerNode(tag string, attrs waBinary.Attrs) *waBinary.Node {
	attrs["call-id"] = "CALL1"
	return &waBinary.Node{Tag: "call", Attrs: waBinary.Attrs{"from": "5511999990000@s.whatsapp.net"},
		Content: []waBinary.Node{{Tag: tag, Attrs: attrs}}}
}

// O celular recusou: o WhatsApp manda <reject>, não <terminate>. O motivo precisa ser "declined"
// (o painel diz "o contato recusou"); antes virava "user_ended" ("o contato desligou").
func TestPeerReject_EndsAsDeclined(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *waBinary.Node
		want core.EndCallReason
	}{
		{"reject sem motivo", peerNode("reject", waBinary.Attrs{"count": "0"}), core.EndCallReasonDeclined},
		{"terminate sem motivo", peerNode("terminate", waBinary.Attrs{}), core.EndCallReasonUserEnded},
		{"terminate com motivo", peerNode("terminate", waBinary.Attrs{"reason": "busy"}), core.EndCallReasonBusy},
		{"reject com motivo", peerNode("reject", waBinary.Attrs{"reason": "busy"}), core.EndCallReasonBusy},
	} {
		t.Run(tc.name, func(t *testing.T) {
			peer := types.NewJID("5511999990000", types.DefaultUserServer)
			m := NewCallManager(&fakeSock{}, slog.Default())
			m.DeferPreaccept = true
			m.HandleCallOffer(context.Background(), offerNode("CALL1", peer.String()), peer)
			var got core.EndCallReason
			m.OnEnded = func(c *CallInfo) { got = c.StateData.EndReason }
			m.HandleCallTerminate(tc.node)
			if got != tc.want {
				t.Fatalf("motivo %q, esperado %q", got, tc.want)
			}
		})
	}
}

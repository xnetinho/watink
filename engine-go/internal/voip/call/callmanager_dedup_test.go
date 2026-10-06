package call

import (
	"bytes"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/media"
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

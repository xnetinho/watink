package calls

import (
	"context"
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// answered devolve uma chamada recebida e atendida, pronta para abrir o áudio.
func answered(t *testing.T) (*rig, *fakeHandle) {
	t.Helper()
	r := newRig(t, false, nil)
	r.s.tickEvery = 20 * time.Millisecond
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	_ = r.s.Accept(context.Background(), callA)
	eventually(t, "accept", func() bool { return r.handle(0).has("accept") })
	return r, r.handle(0)
}

func pcm(n int, v float32) []float32 {
	out := make([]float32, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func TestAudio_PeerPCMArrivesAs640ByteFrames(t *testing.T) {
	r, h := answered(t)
	p, err := r.s.OpenAudio(callA)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	h.media.OnPeerPCM(pcm(960, 0.25))
	for i, want := 0, 3; i < want; i++ {
		select {
		case f := <-p.Out():
			if len(f) != frameBytes {
				t.Fatalf("quadro %d tem %d bytes, esperava 640", i, len(f))
			}
		case <-time.After(time.Second):
			t.Fatalf("faltou o quadro %d (960 amostras = 3 quadros de 320)", i)
		}
	}
}

func TestAudio_OperatorPCMReachesTheCodec(t *testing.T) {
	r, h := answered(t)
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()

	b := make([]byte, 640)
	for i := 0; i < 320; i++ {
		b[i*2], b[i*2+1] = 0x00, 0x40 // 0x4000 = 16384 → 0.5
	}
	p.WritePCM(b)
	p.WritePCM([]byte{1})
	p.WritePCM(nil)
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.fed) != 320 || math.Abs(float64(h.fed[0])-0.5) > 1e-3 {
		t.Fatalf("o codec recebeu %d amostras, primeira=%v", len(h.fed), h.fed)
	}
}

func TestAudio_SlowConsumerNeverBlocksAndQueueIsBounded(t *testing.T) {
	r, h := answered(t)
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 5000; i++ {
			h.media.OnPeerPCM(pcm(320, 0.1))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("o relay bloqueou por causa de um consumidor lento")
	}
	if p.QueueLen() > outQueueFrames {
		t.Fatalf("a fila cresceu sem limite: %d", p.QueueLen())
	}
	if p.dropped.Load() < 4900-outQueueFrames {
		t.Fatalf("o excedente deveria ter sido descartado: %d", p.dropped.Load())
	}
}

func TestAudio_OnlyOnePipePerCall(t *testing.T) {
	r, _ := answered(t)
	p, err := r.s.OpenAudio(callA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.s.OpenAudio(callA); err != ErrAudioInUse {
		t.Fatalf("segundo canal: %v", err)
	}
	p.Close()
	p2, err := r.s.OpenAudio(callA)
	if err != nil {
		t.Fatalf("depois de fechar deve poder reabrir: %v", err)
	}
	p2.Close()
	if _, err := r.s.OpenAudio("NAOEXISTE"); err != ErrNoCall {
		t.Fatal(err)
	}
}

func TestTelemetry_FormatAndValues(t *testing.T) {
	r, h := answered(t)
	h.mu.Lock()
	h.rtt, h.relayUp = 42, true
	h.mu.Unlock()
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()

	now := time.Now()
	for i := 0; i < 100; i++ {
		if i%10 != 5 {
			h.media.OnPeerRtp(uint16(i), uint32(i)*960, 120)
		}
	}
	_ = now
	h.media.OnPeerPCM(pcm(320, 0.5))
	h.media.OnSentRtp(200)
	p.WritePCM(make([]byte, 640))

	var tel Telemetry
	eventually(t, "telemetria com perda", func() bool {
		select {
		case tel = <-p.Telemetry():
			return tel.LossPct > 5
		default:
			return false
		}
	})
	if tel.Type != "quality" || tel.CallID != callA || tel.RttMs == nil || *tel.RttMs != 42 || !tel.RelayConnected {
		t.Fatalf("telemetria: %+v", tel)
	}
	if math.Abs(tel.LossPct-10) > 1 {
		t.Fatalf("perda=%v, esperava ~10%%", tel.LossPct)
	}
}

func TestTelemetry_RttAbsentIsNull(t *testing.T) {
	r, _ := answered(t)
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()
	select {
	case tel := <-p.Telemetry():
		if tel.RttMs != nil {
			t.Fatalf("sem medida o RTT deve ser nulo: %v", *tel.RttMs)
		}
	case <-time.After(time.Second):
		t.Fatal("sem telemetria")
	}
}

func TestTelemetry_PublishedAsCallQualityEvent(t *testing.T) {
	r, h := answered(t)
	h.mu.Lock()
	h.rtt = 30
	h.mu.Unlock()
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()
	eventually(t, "call.quality", func() bool { return len(r.evs("call.quality")) > 0 })
	e := r.evs("call.quality")[0]
	if e["callId"] != callA || e["sessionId"] != "7" {
		t.Fatalf("payload: %v", e)
	}
	if _, has := e["type"]; has {
		t.Fatal("o evento não repete o campo type do WebSocket")
	}
	for _, k := range []string{"lossPct", "jitterMs", "txBytesPerSec", "rxBytesPerSec", "txLevel", "rxLevel", "relayConnected", "relayDrops", "silentMs", "noPeerAudio", "rttMs", "droppedFrames", "at"} {
		if _, ok := e[k]; !ok {
			t.Fatalf("campo %q ausente: %v", k, e)
		}
	}
}

func TestTelemetry_NoPeerAudioAlert(t *testing.T) {
	r, h := answered(t)
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()
	h.media.OnPeerRtp(1, 960, 100)
	eventually(t, "silêncio observado", func() bool {
		select {
		case tel := <-p.Telemetry():
			return tel.SilentMs >= 0 && !tel.NoPeerAudio
		default:
			return false
		}
	})
	r.s.mu.Lock()
	r.s.active.rx.mu.Lock()
	r.s.active.rx.lastArrival = time.Now().Add(-6 * time.Second)
	r.s.active.rx.mu.Unlock()
	r.s.mu.Unlock()
	eventually(t, "alerta de sem áudio", func() bool {
		select {
		case tel := <-p.Telemetry():
			return tel.NoPeerAudio && tel.SilentMs >= 5000
		default:
			return false
		}
	})
}

func TestTelemetry_CountsRelayDrops(t *testing.T) {
	r, h := answered(t)
	h.mu.Lock()
	h.relayUp = true
	h.mu.Unlock()
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()
	eventually(t, "relay up visto", func() bool {
		select {
		case tel := <-p.Telemetry():
			return tel.RelayConnected
		default:
			return false
		}
	})
	h.mu.Lock()
	h.relayUp = false
	h.mu.Unlock()
	eventually(t, "queda contada", func() bool {
		select {
		case tel := <-p.Telemetry():
			return tel.RelayDrops >= 1 && !tel.RelayConnected
		default:
			return false
		}
	})
}

func waitDone(t *testing.T, p *AudioPipe) {
	t.Helper()
	select {
	case <-p.Done():
	case <-time.After(time.Second):
		t.Fatal("o canal de áudio não foi fechado ao encerrar a chamada")
	}
}

func waitGoroutines(t *testing.T, base int) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if runtime.NumGoroutine() <= base+2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, true)
	t.Fatalf("vazamento de goroutines: base=%d agora=%d\n%s", base, runtime.NumGoroutine(), buf[:n])
}

func TestCleanup_NormalEndLeaksNothing(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
	base := runtime.NumGoroutine()
	for i := 0; i < 5; i++ {
		r, _ := answered(t)
		p, _ := r.s.OpenAudio(callA)
		_ = r.s.End(context.Background(), callA)
		eventually(t, "encerrada", func() bool { return len(r.evs("call.ended")) == 1 })
		select {
		case <-p.Done():
		case <-time.After(time.Second):
			t.Fatal("o canal de áudio não foi fechado ao encerrar")
		}
	}
	waitGoroutines(t, base)
}

func TestCleanup_PeerHangupAndFailureLeakNothing(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
	base := runtime.NumGoroutine()
	for i := 0; i < 5; i++ {
		r, _ := answered(t)
		p, _ := r.s.OpenAudio(callA)
		r.s.OnTerminate(&events.CallTerminate{BasicCallMeta: types.BasicCallMeta{From: pn("5511999990001"), CallID: callA}, Data: nil})
		eventually(t, "encerrada pelo contato", func() bool { return len(r.evs("call.ended")) == 1 })
		waitDone(t, p)
	}
	for i := 0; i < 5; i++ {
		r := newRig(t, false, nil)
		r.s.tickEvery = 20 * time.Millisecond
		r.s.newHandle = func(core.VoipSocket) Handle { return &fakeHandle{panicOn: "accept"} }
		r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
		_ = r.s.Ready(context.Background(), callA)
		p, _ := r.s.OpenAudio(callA)
		_ = r.s.Accept(context.Background(), callA)
		eventually(t, "falha", func() bool { return len(r.evs("call.ended")) == 1 })
		waitDone(t, p)
	}
	waitGoroutines(t, base)
}

func TestCleanup_WebSocketDropEndsCallAfterGrace(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
	base := runtime.NumGoroutine()
	r, h := answered(t)
	p, _ := r.s.OpenAudio(callA)
	p.Close()
	if r.clk.fire(audioGrace) != 1 {
		t.Fatal("a rede de segurança de 30 s não foi armada")
	}
	eventually(t, "encerrada", func() bool { return h.has("end:failed") })
	eventually(t, "call.ended", func() bool { return len(r.evs("call.ended")) == 1 })
	waitGoroutines(t, base)
}

func TestCleanup_ReopeningCancelsGrace(t *testing.T) {
	r, _ := answered(t)
	p, _ := r.s.OpenAudio(callA)
	p.Close()
	p2, err := r.s.OpenAudio(callA)
	if err != nil {
		t.Fatal(err)
	}
	defer p2.Close()
	if n := r.clk.fire(audioGrace); n != 0 {
		t.Fatalf("reabrir o canal cancela a rede de segurança (disparou %d)", n)
	}
}

func TestLoad_ReportsActiveCallAndQueuedAudio(t *testing.T) {
	r := newRig(t, false, nil)
	if a, q := r.s.Load(); a != 0 || q != 0 {
		t.Fatalf("sem chamada: %d %d", a, q)
	}
	r, h := answered(t)
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()
	h.media.OnPeerPCM(pcm(320*7, 0.2))
	if a, q := r.s.Load(); a != 1 || q != 7 {
		t.Fatalf("ativa=%d fila=%d, esperava 1 e 7", a, q)
	}
}

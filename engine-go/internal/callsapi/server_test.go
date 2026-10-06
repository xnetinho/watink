package callsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/calls"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/coder/websocket"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	callID = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	token  = "segredo-interno"
)

type fakeHandle struct {
	media calls.MediaHooks
	fed   chan int
}

func (h *fakeHandle) SetHooks(calls.Hooks)                                       {}
func (h *fakeHandle) HandleOffer(context.Context, *waBinary.Node, types.JID)     {}
func (h *fakeHandle) HandleAccept(context.Context, *waBinary.Node, types.JID)    {}
func (h *fakeHandle) HandleTransport(context.Context, *waBinary.Node, types.JID) {}
func (h *fakeHandle) HandleTerminate(*waBinary.Node)                             {}
func (h *fakeHandle) HandleRelayLatency(*waBinary.Node)                          {}
func (h *fakeHandle) SendPreaccept(context.Context) error                        { return nil }
func (h *fakeHandle) Accept(context.Context, string) error                       { return nil }
func (h *fakeHandle) Reject(context.Context, string) error                       { return nil }
func (h *fakeHandle) End(context.Context, string) error                          { return nil }
func (h *fakeHandle) Start(context.Context, string, types.JID) error             { return nil }
func (h *fakeHandle) Abandon(string)                                             {}
func (h *fakeHandle) SetMedia(k calls.MediaHooks)                                { h.media = k }
func (h *fakeHandle) FeedPCM(p []float32)                                        { h.fed <- len(p) }
func (h *fakeHandle) RelayRTTMs() (int, bool)                                    { return 25, true }
func (h *fakeHandle) RelayConnected() bool                                       { return true }

type backend struct{ s *calls.Session }

func (b backend) OpenCallAudio(id string) (*calls.AudioPipe, error) { return b.s.OpenAudio(id) }
func (b backend) CallsLoad() (int, int)                             { return b.s.Load() }

func setup(t *testing.T) (*httptest.Server, *fakeHandle, *calls.Session) {
	t.Helper()
	h := &fakeHandle{fed: make(chan int, 16)}
	sess := calls.NewSession(calls.SessionConfig{
		ID: 1, TenantID: "t", TickEvery: 30 * time.Millisecond,
		Publish:   func(string, int, string, map[string]interface{}) {},
		NewHandle: func(core.VoipSocket) calls.Handle { return h },
	})
	from := types.NewJID("5511999990001", types.DefaultUserServer)
	sess.OnOffer(context.Background(), &events.CallOffer{
		BasicCallMeta: types.BasicCallMeta{From: from, CallCreator: from, CallID: callID},
		Data:          &waBinary.Node{Tag: "offer"},
	})
	_ = sess.Ready(context.Background(), callID)
	_ = sess.Accept(context.Background(), callID)
	srv := httptest.NewServer(NewHandler(backend{sess}, token))
	t.Cleanup(srv.Close)
	return srv, h, sess
}

func dial(t *testing.T, srv *httptest.Server, path, tok string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	hdr := http.Header{}
	if tok != "" {
		hdr.Set("X-Internal-Token", tok)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+path, &websocket.DialOptions{HTTPHeader: hdr})
}

func TestWithoutToken_ServerDoesNotStart(t *testing.T) {
	t.Setenv("CALLS_AUDIO_TOKEN", "")
	if Token() != "" {
		t.Fatal("token vazio")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { Start(ctx, backend{}); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sem CALLS_AUDIO_TOKEN o servidor deve retornar na hora, sem subir")
	}
	cancel()
}

func TestWrongOrMissingToken_Is401(t *testing.T) {
	srv, _, _ := setup(t)
	for _, tok := range []string{"", "errado", token + "x", token[:len(token)-1]} {
		_, resp, err := dial(t, srv, "/calls/"+callID+"/audio", tok)
		if err == nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("token %q: err=%v resp=%v", tok, err, resp)
		}
	}
}

func TestInvalidAndUnknownCall(t *testing.T) {
	srv, _, _ := setup(t)
	_, resp, err := dial(t, srv, "/calls/../etc/audio", token)
	if err == nil || resp == nil || resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("id inválido: %v %v", err, resp)
	}
	_, resp, err = dial(t, srv, "/calls/ZZZ/audio", token)
	if err == nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("id mal formado: %v", resp)
	}
	_, resp, err = dial(t, srv, "/calls/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB/audio", token)
	if err == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("chamada inexistente: %v", resp)
	}
}

func TestAudioFlowsBothWaysAndTelemetryArrives(t *testing.T) {
	srv, h, _ := setup(t)
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio", token)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := c.Write(ctx, websocket.MessageBinary, make([]byte, 640)); err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-h.fed:
		if n != 320 {
			t.Fatalf("o codec recebeu %d amostras", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("áudio do operador não chegou ao codec")
	}

	pcm := make([]float32, 320)
	for i := range pcm {
		pcm[i] = 0.3
	}
	h.media.OnPeerPCM(pcm)

	var gotAudio, gotTel bool
	for !(gotAudio && gotTel) {
		typ, data, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("leitura: %v (áudio=%v telemetria=%v)", err, gotAudio, gotTel)
		}
		switch typ {
		case websocket.MessageBinary:
			if len(data) != 640 {
				t.Fatalf("quadro de %d bytes", len(data))
			}
			gotAudio = true
		case websocket.MessageText:
			var tel calls.Telemetry
			if err := json.Unmarshal(data, &tel); err != nil || tel.Type != "quality" || tel.CallID != callID || tel.RttMs == nil || *tel.RttMs != 25 {
				t.Fatalf("telemetria inválida: %s err=%v", data, err)
			}
			gotTel = true
		}
	}
}

func TestSecondConnectionIs409(t *testing.T) {
	srv, _, _ := setup(t)
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio", token)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	_, resp, err := dial(t, srv, "/calls/"+callID+"/audio", token)
	if err == nil || resp == nil || resp.StatusCode != http.StatusConflict {
		t.Fatalf("segunda conexão: err=%v resp=%v", err, resp)
	}
}

func TestClientDropReleasesPipe(t *testing.T) {
	srv, _, sess := setup(t)
	base := runtime.NumGoroutine()
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio", token)
	if err != nil {
		t.Fatal(err)
	}
	c.CloseNow()
	deadline := time.Now().Add(2 * time.Second)
	for {
		c2, _, err := dial(t, srv, "/calls/"+callID+"/audio", token)
		if err == nil {
			c2.CloseNow()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("o canal não foi liberado depois da queda do WebSocket")
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = sess
	time.Sleep(200 * time.Millisecond)
	if g := runtime.NumGoroutine(); g > base+6 {
		t.Fatalf("goroutines acumulando: base=%d agora=%d", base, g)
	}
}

func TestCallEndClosesWebSocket(t *testing.T) {
	srv, _, sess := setup(t)
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio", token)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	sess.AbandonAll(calls.ReasonInterrupted)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		if _, _, err := c.Read(ctx); err != nil {
			if ctx.Err() != nil {
				t.Fatal("o WebSocket não fechou quando a chamada acabou")
			}
			return
		}
	}
}

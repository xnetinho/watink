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

const callID = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

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
	srv := httptest.NewServer(NewHandler(backend{sess}))
	t.Cleanup(srv.Close)
	return srv, h, sess
}

func dial(t *testing.T, srv *httptest.Server, path string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+path, nil)
}

// O canal é interno e não tem credencial: a defesa é a rede (porta só em expose). O teste
// trava o contrato para ninguém reintroduzir, sem querer, um cabeçalho obrigatório que o
// business não manda.
func TestAudioDoesNotRequireCredentials(t *testing.T) {
	srv, _, _ := setup(t)
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio")
	if err != nil {
		t.Fatalf("sem credencial deveria conectar: %v", err)
	}
	_ = c.CloseNow()
}

func TestInvalidAndUnknownCall(t *testing.T) {
	srv, _, _ := setup(t)
	_, resp, err := dial(t, srv, "/calls/../etc/audio")
	if err == nil || resp == nil || resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("id inválido: %v %v", err, resp)
	}
	_, resp, err = dial(t, srv, "/calls/ZZZ/audio")
	if err == nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("id mal formado: %v", resp)
	}
	_, resp, err = dial(t, srv, "/calls/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB/audio")
	if err == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("chamada inexistente: %v", resp)
	}
}

func TestAudioFlowsBothWaysAndTelemetryArrives(t *testing.T) {
	srv, h, _ := setup(t)
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio")
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
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	_, resp, err := dial(t, srv, "/calls/"+callID+"/audio")
	if err == nil || resp == nil || resp.StatusCode != http.StatusConflict {
		t.Fatalf("segunda conexão: err=%v resp=%v", err, resp)
	}
}

func TestClientDropReleasesPipe(t *testing.T) {
	srv, _, sess := setup(t)
	base := runtime.NumGoroutine()
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio")
	if err != nil {
		t.Fatal(err)
	}
	c.CloseNow()
	deadline := time.Now().Add(2 * time.Second)
	for {
		c2, _, err := dial(t, srv, "/calls/"+callID+"/audio")
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
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio")
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

// O quadro de vídeo do contato sai no mesmo WebSocket, com o prefixo mágico, e o áudio não é afetado.
func TestAudioSocketCarriesVideoFrames(t *testing.T) {
	srv, h, _ := setup(t)
	c, _, err := dial(t, srv, "/calls/"+callID+"/audio")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	deadline := time.Now().Add(2 * time.Second)
	for h.media.OnPeerVideoFrame == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if h.media.OnPeerVideoFrame == nil {
		t.Fatal("o gancho de vídeo não foi ligado")
	}
	au := []byte{0, 0, 0, 1, 0x65, 9, 9, 9}
	h.media.OnPeerVideoFrame(au, true, 3)
	h.media.OnPeerPCM(make([]float32, 320))

	gotVideo, gotAudio := false, false
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for i := 0; i < 4 && !(gotVideo && gotAudio); i++ {
		typ, msg, err := c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if typ != websocket.MessageBinary {
			continue
		}
		if calls.IsVideoFrame(msg) {
			_, key, rot, body, derr := calls.DecodeVideoFrame(msg)
			if derr != nil || !key || rot != 3 || string(body) != string(au) {
				t.Fatalf("quadro de vídeo adulterado: key=%v rot=%d body=%x err=%v", key, rot, body, derr)
			}
			gotVideo = true
		} else if len(msg) == 640 {
			gotAudio = true
		}
	}
	if !gotVideo || !gotAudio {
		t.Fatalf("vídeo=%v áudio=%v: os dois tinham de chegar", gotVideo, gotAudio)
	}
}

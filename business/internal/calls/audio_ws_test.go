package calls

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEngine é um servidor WebSocket que imita o endpoint interno do engine.
type fakeEngine struct {
	srv      *httptest.Server
	mu       sync.Mutex
	gotAudio [][]byte
	conn     chan *websocket.Conn
	hits     int
}

func newFakeEngine(t *testing.T) *fakeEngine {
	t.Helper()
	f := &fakeEngine{conn: make(chan *websocket.Conn, 4)}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.hits++
		f.mu.Unlock()
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		f.conn <- c
		for {
			typ, data, err := c.Read(r.Context())
			if err != nil {
				return
			}
			if typ == websocket.MessageBinary {
				f.mu.Lock()
				f.gotAudio = append(f.gotAudio, data)
				f.mu.Unlock()
			}
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeEngine) base() string { return "ws" + strings.TrimPrefix(f.srv.URL, "http") }

func (f *fakeEngine) audio() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.gotAudio...)
}

// browserEndpoint expõe ServeAudio como o handler HTTP faria, para o teste abrir
// um WebSocket "de navegador" de verdade.
func browserEndpoint(t *testing.T, r *rig, a *Audio, dial EngineDialer, userID int, callID string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := r.svc.AuthorizeAudio(r.tenant, userID, callID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		c, err := websocket.Accept(w, req, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		r.svc.ServeAudio(req.Context(), a, dial, c, r.tenant, userID, callID)
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
}

func answeredCall(t *testing.T, r *rig, callID string) int {
	t.Helper()
	ringing(t, r, callID)
	uid := r.users["da_fila_A"].ID
	_, err := r.svc.Accept(ctx, r.tenant, uid, callID)
	require.NoError(t, err)
	return uid
}

// 7.4: os bytes chegam íntegros nos dois sentidos.
func TestServeAudio_BytesArriveIntactBothWays(t *testing.T) {
	r := newRig(t)
	uid := answeredCall(t, r, "AU-1")
	eng := newFakeEngine(t)
	a := NewAudio()

	browser, _, err := browserEndpoint(t, r, a, NewEngineDialer(eng.base()), uid, "AU-1")
	require.NoError(t, err)
	defer browser.CloseNow()
	engConn := <-eng.conn

	up := make([]byte, FrameBytes)
	for i := range up {
		up[i] = byte(i % 251)
	}
	require.NoError(t, browser.Write(ctx, websocket.MessageBinary, up))
	assert.Eventually(t, func() bool { a := eng.audio(); return len(a) == 1 && bytes.Equal(a[0], up) }, 2*time.Second, 10*time.Millisecond,
		"o quadro do operador chega ao engine sem alteração")

	down := bytes.Repeat([]byte{0xAB, 0xCD}, FrameBytes/2)
	require.NoError(t, engConn.Write(ctx, websocket.MessageBinary, down))
	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	typ, got, err := browser.Read(rctx)
	require.NoError(t, err)
	assert.Equal(t, websocket.MessageBinary, typ)
	assert.Equal(t, down, got, "o quadro do contato chega ao operador sem alteração")
}

func TestServeAudio_TelemetryGoesToHandlingOperatorOnly(t *testing.T) {
	r := newRig(t)
	uid := answeredCall(t, r, "AU-2")
	eng := newFakeEngine(t)
	a := NewAudio()
	browser, _, err := browserEndpoint(t, r, a, NewEngineDialer(eng.base()), uid, "AU-2")
	require.NoError(t, err)
	defer browser.CloseNow()
	engConn := <-eng.conn

	tel, _ := json.Marshal(map[string]interface{}{"type": "quality", "callId": "AU-2", "rttMs": 40.0, "lossPct": 7.0, "jitterMs": 5.0})
	require.NoError(t, engConn.Write(ctx, websocket.MessageText, tel))
	assert.Eventually(t, func() bool { return r.bc.to(UserRoom(r.tenant, uid), "call.quality") == 1 }, 2*time.Second, 10*time.Millisecond)
	assert.Zero(t, r.bc.to("tenant:"+r.tenant.String(), "call.quality"), "a empresa toda não recebe")
}

func TestAuthorizeAudio_Rules(t *testing.T) {
	r := newRig(t)
	uid := answeredCall(t, r, "AU-3")
	other := r.users["setor_fila_A"].ID

	assert.NoError(t, r.svc.AuthorizeAudio(r.tenant, uid, "AU-3"))
	assert.Equal(t, ErrNotYourCall, r.svc.AuthorizeAudio(r.tenant, other, "AU-3"), "outro operador da mesma empresa")
	assert.Equal(t, ErrNotFound, r.svc.AuthorizeAudio(r.other, r.users["de_outra_empresa"].ID, "AU-3"), "outra empresa: não revela que existe")
	assert.Equal(t, ErrNotFound, r.svc.AuthorizeAudio(r.tenant, uid, "NAO-EXISTE"))

	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, ended("AU-3", "user_ended", 5)))
	assert.Equal(t, ErrNotActive, r.svc.AuthorizeAudio(r.tenant, uid, "AU-3"), "chamada encerrada não abre áudio")
}

func TestAuthorizeAudio_RingingCallNotYetAnsweredIsDenied(t *testing.T) {
	r := newRig(t)
	ringing(t, r, "AU-4")
	assert.Equal(t, ErrNotYourCall, r.svc.AuthorizeAudio(r.tenant, r.users["da_fila_A"].ID, "AU-4"), "quem não assumiu não abre áudio")
}

func TestServeAudio_SecondBrowserIsRejected(t *testing.T) {
	r := newRig(t)
	uid := answeredCall(t, r, "AU-5")
	eng := newFakeEngine(t)
	a := NewAudio()
	first, _, err := browserEndpoint(t, r, a, NewEngineDialer(eng.base()), uid, "AU-5")
	require.NoError(t, err)
	defer first.CloseNow()
	<-eng.conn

	second, _, err := browserEndpoint(t, r, a, NewEngineDialer(eng.base()), uid, "AU-5")
	require.NoError(t, err)
	defer second.CloseNow()
	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, _, err = second.Read(rctx)
	require.Error(t, err, "o segundo canal é fechado")
	assert.Equal(t, websocket.StatusPolicyViolation, websocket.CloseStatus(err))
	assert.Equal(t, 1, a.Count(), "o primeiro segue intacto")
}

func TestServeAudio_EngineUnavailableClosesAndEndsCall(t *testing.T) {
	r := newRig(t)
	uid := answeredCall(t, r, "AU-6")
	a := NewAudio()
	bad := NewEngineDialer("ws://127.0.0.1:1")
	browser, _, err := browserEndpoint(t, r, a, bad, uid, "AU-6")
	require.NoError(t, err)
	defer browser.CloseNow()
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, _, err = browser.Read(rctx)
	require.Error(t, err)
	assert.Equal(t, websocket.StatusInternalError, websocket.CloseStatus(err))
	var ce websocket.CloseError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, "audio_unavailable", ce.Reason, "o motivo chega ao navegador para o painel explicar a falha")
	assert.Eventually(t, func() bool { return len(r.pub.cmds("call.end")) == 1 }, 2*time.Second, 10*time.Millisecond,
		"sem áudio a chamada é encerrada (o contato não fica pendurado)")
	assert.Eventually(t, func() bool { return a.Count() == 0 }, 2*time.Second, 10*time.Millisecond, "nenhuma ponte vaza")
}

func TestEngineDialer_WithoutAddressFailsWithClearError(t *testing.T) {
	_, err := NewEngineDialer("")(ctx, "X")
	assert.ErrorIs(t, err, ErrAudioNotConfigured)
}

// 7.5 (integração): o navegador cai e não volta → a chamada é encerrada.
func TestServeAudio_BrowserDropEndsCallAfterGrace(t *testing.T) {
	r := newRig(t)
	uid := answeredCall(t, r, "AU-7")
	eng := newFakeEngine(t)
	a := NewAudio()
	a.dropTO = 80 * time.Millisecond
	browser, _, err := browserEndpoint(t, r, a, NewEngineDialer(eng.base()), uid, "AU-7")
	require.NoError(t, err)
	<-eng.conn
	browser.CloseNow()

	assert.Eventually(t, func() bool { return len(r.pub.cmds("call.end")) == 1 }, 3*time.Second, 20*time.Millisecond,
		"sem o canal além do prazo, o business manda o engine desligar")
}

// 7.6 (integração): um navegador que não lê não pode travar o engine nem fazer a
// memória crescer. Aqui o engine despeja áudio sem parar enquanto o navegador não
// lê; o engine nunca trava e a fila para o navegador fica no limite. (Quanto é
// descartado depende dos buffers TCP do SO; o descarte em si é provado, sem rede,
// por TestPipe_SlowConsumerIsBoundedAndProducerNeverBlocks.)
func TestServeAudio_SlowBrowserNeverBlocksEngineAndQueueStaysBounded(t *testing.T) {
	r := newRig(t)
	uid := answeredCall(t, r, "AU-8")
	eng := newFakeEngine(t)
	a := NewAudio()

	browser, _, err := browserEndpoint(t, r, a, NewEngineDialer(eng.base()), uid, "AU-8")
	require.NoError(t, err)
	defer browser.CloseNow()
	engConn := <-eng.conn
	var bridge *Bridge
	require.Eventually(t, func() bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		bridge = a.bridges["AU-8"]
		return bridge != nil
	}, 2*time.Second, 10*time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 3000; i++ {
			wctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := engConn.Write(wctx, websocket.MessageBinary, frame(byte(i)))
			cancel()
			if err != nil {
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("o engine travou escrevendo para um navegador lento")
	}
	assert.LessOrEqual(t, bridge.ToBrowser.Len(), bridgeQueue, "a fila para o navegador nunca passa do limite")
}

// ---- vídeo no canal (fase 1) ----

// videoMsg monta um quadro de vídeo no formato do canal (prefixo FF 56 44 01 + flags + ts + Annex-B).
func videoMsg(key bool, au []byte) []byte {
	out := []byte{0xFF, 'V', 'D', 0x01, 0, 0, 0, 0, 0}
	if key {
		out[4] = 0x01
	}
	return append(out, au...)
}

// O vídeo do contato atravessa o business até o navegador sem alteração, e o áudio continua igual.
func TestServeAudio_VideoFromEngineReachesBrowserUnchanged(t *testing.T) {
	r := newRig(t)
	uid := answeredCall(t, r, "VID-1")
	eng := newFakeEngine(t)
	browser, _, err := browserEndpoint(t, r, NewAudio(), NewEngineDialer(eng.base()), uid, "VID-1")
	require.NoError(t, err)
	defer browser.CloseNow()
	engConn := <-eng.conn

	au := bytes.Repeat([]byte{0x65, 0xAB}, 1500) // 3000 B: bem maior que um quadro de PCM
	msg := videoMsg(true, au)
	require.NoError(t, engConn.Write(ctx, websocket.MessageBinary, msg))

	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, got, err := browser.Read(rctx)
	require.NoError(t, err)
	assert.Equal(t, msg, got, "o quadro de vídeo chega inteiro ao navegador")
}

// O gravador de ÁUDIO nunca pode receber bytes de vídeo: o Tap é só de PCM. 60 quadros de vídeo de 6 KB
// somam ~360 KB; lidos como PCM seriam ~11 s de "áudio" (32 KB/s), e a duração gravada sairia > 0.
func TestServeAudio_VideoNeverReachesTheAudioRecorder(t *testing.T) {
	r := newRig(t)
	r.withRecording(t, newMemStore())
	r.setMode(t, "auto")
	uid := answeredCall(t, r, "VID-2")
	require.NoError(t, r.svc.HandleState(ctx, r.tenant, stateEvent("VID-2", "active")))
	eng := newFakeEngine(t)
	browser, _, err := browserEndpoint(t, r, NewAudio(), NewEngineDialer(eng.base()), uid, "VID-2")
	require.NoError(t, err)
	defer browser.CloseNow()
	engConn := <-eng.conn
	require.True(t, r.svc.Recording().Active(r.tenant, "VID-2"))

	const n = 60
	for i := 0; i < n; i++ {
		require.NoError(t, engConn.Write(ctx, websocket.MessageBinary, videoMsg(i == 0, bytes.Repeat([]byte{0x41}, 6000))))
	}
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	for i := 0; i < n; i++ {
		_, _, err := browser.Read(rctx)
		require.NoError(t, err)
	}
	time.Sleep(600 * time.Millisecond)

	raw, _ := json.Marshal(map[string]interface{}{"callId": "VID-2", "endReason": "user_ended", "durationSecs": 1, "direction": "incoming"})
	require.NoError(t, r.svc.HandleEnded(ctx, r.tenant, raw))
	assert.Zero(t, r.log(t, "VID-2").RecordingDurationSec, "o vídeo não pode virar áudio gravado")
}

// O vídeo e o PCM não se misturam: PCM do operador continua indo ao engine, e um quadro de vídeo vindo do
// NAVEGADOR (fase 2, ainda não suportada) é descartado em vez de ser entregue como se fosse áudio.
func TestServeAudio_VideoFromBrowserIsNotForwardedAsAudio(t *testing.T) {
	r := newRig(t)
	uid := answeredCall(t, r, "VID-3")
	eng := newFakeEngine(t)
	browser, _, err := browserEndpoint(t, r, NewAudio(), NewEngineDialer(eng.base()), uid, "VID-3")
	require.NoError(t, err)
	defer browser.CloseNow()
	<-eng.conn

	require.NoError(t, browser.Write(ctx, websocket.MessageBinary, videoMsg(true, bytes.Repeat([]byte{1}, 500))))
	pcm := make([]byte, FrameBytes)
	require.NoError(t, browser.Write(ctx, websocket.MessageBinary, pcm))
	assert.Eventually(t, func() bool { return len(eng.audio()) == 1 }, 2*time.Second, 10*time.Millisecond)
	assert.Len(t, eng.audio(), 1, "só o PCM chegou ao engine; o vídeo do navegador não vira áudio")
}

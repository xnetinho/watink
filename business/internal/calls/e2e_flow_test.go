package calls

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/recording"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/coder/websocket"
	gomp3 "github.com/hajimehoshi/go-mp3"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Teste de fluxo completo SEM WhatsApp (tarefa 10.3), com RabbitMQ e Postgres REAIS:
//
//	oferta → elegibilidade → atender → áudio → telemetria → encerrar → CallLogs + gravação
//
// O business roda de verdade (RabbitMQService, consumidor dedicado, Service, ServeAudio,
// gravador, MP3). O lado do ENGINE é simulado por um cliente AMQP e um WebSocket reais
// que falam o contrato exato (eventos wbot.<t>.<s>.call.*, comandos engine.go.calls e o
// canal interno de áudio). Assim o que se prova é a fronteira, não um mock da fronteira.
//
// Rode com: AMQP_TEST_URL=amqp://guest:guest@localhost:56720/ go test ./internal/calls -run E2E
func TestE2E_FullCallFlowOverRealRabbitMQ(t *testing.T) {
	url := os.Getenv("AMQP_TEST_URL")
	if url == "" {
		t.Skip("AMQP_TEST_URL não definido (precisa de um RabbitMQ real)")
	}
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")
	r.grant(t, "da_fila_A", "read")
	r.online("da_fila_A")
	uid := r.users["da_fila_A"].ID
	store := newMemStore()
	r.setMode(t, "optional")

	// --- business: RabbitMQ real + serviço real, ligados como no main.go ---
	t.Setenv("AMQP_URL", url)
	bus := services.NewRabbitMQProvider(url)
	require.NoError(t, bus.Connect())
	defer bus.Close()
	svc := NewService(r.db, repository.NewGORMContactRepo(r.db), repository.NewGORMTicketRepo(r.db),
		repository.NewGORMQueueRepo(r.db), bus, r.bc, r.pres).WithRecording(NewRecording(store, t.TempDir()))
	svc.now = time.Now
	// fila própria por execução (não colide com outra rodada nem com o ambiente)
	suffix := time.Now().Format("150405.000000")
	require.NoError(t, bus.ConsumeEvents("e2e.calls."+suffix, CallEventRoutingKeys, func(d amqp.Delivery) error {
		return svc.Dispatch(context.Background(), d.Body)
	}))

	// --- engine simulado: cliente AMQP real ---
	conn, err := amqp.Dial(url)
	require.NoError(t, err)
	defer conn.Close()
	ch, err := conn.Channel()
	require.NoError(t, err)
	var cmdMu sync.Mutex
	var cmds []string
	cmdQ, err := ch.QueueDeclare("e2e.engine.calls."+suffix, false, true, false, false, nil)
	require.NoError(t, err)
	// o exchange de comandos é declarado pelo business no Connect(); aqui só se liga a fila
	require.NoError(t, ch.QueueBind(cmdQ.Name, "wbot.*.*.call.*", "wbot.commands", false, nil), "ligar a fila de comandos do engine simulado")
	msgs, err := ch.Consume(cmdQ.Name, "", true, false, false, false, nil)
	require.NoError(t, err)
	go func() {
		for d := range msgs {
			cmdMu.Lock()
			cmds = append(cmds, d.RoutingKey)
			cmdMu.Unlock()
		}
	}()
	gotCmd := func(want string) bool {
		cmdMu.Lock()
		defer cmdMu.Unlock()
		for _, c := range cmds {
			if strings.HasSuffix(c, want) {
				return true
			}
		}
		return false
	}
	publish := func(typ string, payload map[string]interface{}) {
		payload["sessionId"] = itoa(r.waA.ID)
		body, _ := json.Marshal(map[string]interface{}{"id": "e2e", "timestamp": time.Now().UnixMilli(), "tenantId": r.tenant.String(), "type": typ, "payload": payload})
		require.NoError(t, ch.Publish("wbot.events", "wbot."+r.tenant.String()+"."+itoa(r.waA.ID)+"."+typ, false, false,
			amqp.Publishing{ContentType: "application/json", Body: body}))
	}

	const callID = "E2E00000000000000000000000000001"

	// 1) oferta do WhatsApp (chega como evento do engine)
	publish("call.incoming", map[string]interface{}{
		"callId": callID, "peer": "5511999990050@s.whatsapp.net", "callerPn": "5511999990050", "direction": "incoming", "media": "audio",
	})
	require.Eventually(t, func() bool { return r.countLogs(t, callID) == 1 }, 5*time.Second, 25*time.Millisecond, "o business registrou a oferta")

	// 2) elegibilidade: confirmou ao engine (call.ready) e tocou só o operador elegível
	require.Eventually(t, func() bool { return gotCmd(".call.ready") }, 5*time.Second, 25*time.Millisecond, "business → engine: call.ready pela fila de comandos real")
	assert.Equal(t, 1, r.bc.to(UserRoom(r.tenant, uid), "call.incoming"), "tocou para o operador elegível")
	assert.Equal(t, StatusRinging, r.log(t, callID).Status)

	// 3) o operador atende → business manda call.accept ao engine
	_, err = svc.Accept(ctx, r.tenant, uid, callID)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return gotCmd(".call.accept") }, 5*time.Second, 25*time.Millisecond, "business → engine: call.accept")
	publish("call.state", map[string]interface{}{"callId": callID, "state": "active", "direction": "incoming"})
	require.Eventually(t, func() bool { return r.log(t, callID).Status == StatusActive }, 5*time.Second, 25*time.Millisecond, "call.state ativa a chamada")

	// 4) gravação opcional ligada pelo operador; áudio sintético pelo canal WebSocket real
	require.NoError(t, svc.StartRecording(r.tenant, uid, callID, false))

	engineConnCh := make(chan *websocket.Conn, 1)
	engineGot := make(chan []byte, 8)
	engineSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		c, err := websocket.Accept(w, req, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		engineConnCh <- c
		for {
			typ, data, err := c.Read(req.Context())
			if err != nil {
				return
			}
			if typ == websocket.MessageBinary {
				select {
				case engineGot <- data:
				default:
				}
			}
		}
	}))
	defer engineSrv.Close()

	audio := NewAudio()
	dial := NewEngineDialer("ws" + strings.TrimPrefix(engineSrv.URL, "http"))
	browserSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := svc.AuthorizeAudio(r.tenant, uid, callID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		c, err := websocket.Accept(w, req, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		svc.ServeAudio(req.Context(), audio, dial, c, r.tenant, uid, callID)
	}))
	defer browserSrv.Close()
	dctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	browser, _, err := websocket.Dial(dctx, "ws"+strings.TrimPrefix(browserSrv.URL, "http"), nil)
	require.NoError(t, err)
	defer browser.CloseNow()
	var engineConn *websocket.Conn
	select {
	case engineConn = <-engineConnCh:
	case <-time.After(5 * time.Second):
		t.Fatal("o business não abriu o canal interno do engine")
	}

	// áudio dos dois sentidos, ~1,5 s, em quadros de 20 ms
	var sentUp [][]byte
	for i := 0; i < 75; i++ {
		up := pcmFrame(8000, 440, i)
		sentUp = append(sentUp, up)
		require.NoError(t, browser.Write(dctx, websocket.MessageBinary, up))
		require.NoError(t, engineConn.Write(dctx, websocket.MessageBinary, pcmFrame(6000, 880, i)))
		time.Sleep(20 * time.Millisecond)
	}
	select {
	case first := <-engineGot:
		assert.Equal(t, sentUp[0], first, "o primeiro quadro do operador chegou ao engine sem alteração")
	case <-time.After(3 * time.Second):
		t.Fatal("nenhum quadro do operador chegou ao engine")
	}
	rctx, rcancel := context.WithTimeout(ctx, 3*time.Second)
	typ, down, err := browser.Read(rctx)
	rcancel()
	require.NoError(t, err)
	assert.Equal(t, websocket.MessageBinary, typ)
	assert.Len(t, down, recording.FrameSamples*2, "o áudio do contato chegou ao navegador em quadro de 640 B")

	// 5) telemetria do engine pelo canal de áudio → só o operador da chamada recebe
	tel, _ := json.Marshal(map[string]interface{}{"type": "quality", "callId": callID, "rttMs": 55.0, "lossPct": 0.5, "jitterMs": 8.0})
	require.NoError(t, engineConn.Write(dctx, websocket.MessageText, tel))
	require.Eventually(t, func() bool { return r.bc.to(UserRoom(r.tenant, uid), "call.quality") >= 1 }, 5*time.Second, 25*time.Millisecond, "telemetria interpretada e entregue ao operador")
	assert.Zero(t, r.bc.to(UserRoom(r.tenant, r.users["setor_fila_A"].ID), "call.quality"))
	assert.Zero(t, r.bc.to("tenant:"+r.tenant.String(), "call.quality"), "e nunca à empresa toda")

	// 6) o contato desliga: call.ended pelo RabbitMQ real
	time.Sleep(300 * time.Millisecond)
	publish("call.ended", map[string]interface{}{"callId": callID, "direction": "incoming", "endReason": "user_ended", "durationSecs": 2})
	require.Eventually(t, func() bool { return r.log(t, callID).EndedAt != nil }, 8*time.Second, 50*time.Millisecond, "call.ended fechou o registro")

	// 7) resultado final: CallLogs completo + gravação no S3 + mensagem no ticket
	l := r.log(t, callID)
	assert.Equal(t, StatusEnded, l.Status)
	assert.Equal(t, 2, l.DurationSec)
	require.NotNil(t, l.HandledByUserID)
	assert.Equal(t, uid, *l.HandledByUserID)
	assert.Equal(t, "user_ended", l.EndReason)
	require.NotNil(t, l.MosEstimated, "o resumo de qualidade foi gravado")
	assert.Equal(t, 1, l.QualitySamples)

	assert.Equal(t, recording.StatusReady, l.RecordingStatus, "a gravação terminou")
	assert.Equal(t, recording.ObjectKey(r.tenant, callID), l.RecordingKey)
	mp3, ok := store.objects[l.RecordingKey]
	require.True(t, ok, "o MP3 está no armazenamento")
	rate, secs, _, rms := decodeMP3(t, mp3)
	assert.Equal(t, 16000, rate)
	assert.InDelta(t, 1.5, secs, 1.5, "duração do MP3 coerente com o áudio enviado")
	assert.Greater(t, rms, 0.01, "a gravação tem o áudio dos dois lados, não silêncio")

	var m models.Message
	require.NoError(t, r.db.Where(`id = ?`, callMessageID(callID)).First(&m).Error)
	assert.Equal(t, "call", m.MediaType, "a chamada ficou no histórico do ticket")
	assert.Equal(t, 1, r.bc.to("tenant:"+r.tenant.String(), "call.ended"))
}

// decodeMP3 usa um decodificador INDEPENDENTE (go-mp3) e devolve taxa, duração, tom e nível.
func decodeMP3(t *testing.T, b []byte) (rate int, secs float64, hz float64, rms float64) {
	t.Helper()
	d, err := gomp3.NewDecoder(bytes.NewReader(b))
	require.NoError(t, err, "o MP3 gravado deve ser válido para um decodificador independente")
	raw, err := io.ReadAll(d)
	require.NoError(t, err)
	n := len(raw) / 4
	var sum float64
	for i := 0; i < n; i++ {
		v := float64(int16(uint16(raw[i*4])|uint16(raw[i*4+1])<<8)) / 32768
		sum += v * v
	}
	if n > 0 {
		rms = math.Sqrt(sum / float64(n))
	}
	return d.SampleRate(), float64(n) / float64(d.SampleRate()), 0, rms
}

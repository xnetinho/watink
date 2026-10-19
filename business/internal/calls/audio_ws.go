package calls

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// EngineDialer abre o WebSocket interno de áudio do engine para uma chamada.
type EngineDialer func(ctx context.Context, callID string) (*websocket.Conn, error)

// NewEngineDialer devolve o discador de produção: base é "ws://engine:8085", montada de
// ENGINE_HOST. O canal é interno (rede do Docker) e não tem autenticação própria: a porta
// do engine nunca pode ser publicada fora da rede. Sem base não há áudio.
func NewEngineDialer(base string) EngineDialer {
	return func(ctx context.Context, callID string) (*websocket.Conn, error) {
		if base == "" {
			return nil, ErrAudioNotConfigured
		}
		c, _, err := websocket.Dial(ctx, base+"/calls/"+callID+"/audio", nil)
		return c, err
	}
}

// ErrAudioNotConfigured: o business não sabe onde está o canal de áudio do engine.
var ErrAudioNotConfigured = errors.New("canal de áudio das chamadas não configurado (defina ENGINE_HOST no business)")

// maxVideoMessage é o maior quadro aceito do navegador: um quadro-chave de 640x480 a 600 kbps passa de 64 KB
// com folga em cenas com movimento.
const maxVideoMessage = 512 * 1024

// cameraCommand é o comando de texto do navegador para ligar/desligar a câmera. O business o valida e
// o reserializa: nunca repassa texto cru do navegador ao engine.
type cameraCommand struct {
	Type        string `json:"type"`
	On          bool   `json:"on"`
	Orientation int    `json:"orientation"`
}

// parseCameraCommand devolve o comando já saneado (orientação fora de 0..3 vira 0). ok=false para qualquer
// outro texto.
func parseCameraCommand(raw []byte) (cameraCommand, bool) {
	var c cameraCommand
	if json.Unmarshal(raw, &c) != nil || c.Type != "camera" {
		return cameraCommand{}, false
	}
	if c.Orientation < 0 || c.Orientation > 3 {
		c.Orientation = 0
	}
	return c, true
}

// isKeyframeRequest diz se o texto do engine é o pedido de quadro-chave para o navegador.
func isKeyframeRequest(raw []byte) bool {
	var c struct {
		Type string `json:"type"`
	}
	return json.Unmarshal(raw, &c) == nil && c.Type == "keyframe"
}

// Telemetry é o que o engine manda como texto no WebSocket de áudio.
type Telemetry struct {
	Type        string   `json:"type"`
	CallID      string   `json:"callId"`
	RttMs       *float64 `json:"rttMs"`
	LossPct     float64  `json:"lossPct"`
	JitterMs    float64  `json:"jitterMs"`
	NoPeerAudio bool     `json:"noPeerAudio"`
	SilentMs    int64    `json:"silentMs"`
}

// ErrNotYourCall: só o operador que assumiu a chamada abre o áudio.
var ErrNotYourCall = errors.New("esta chamada não é sua")

// AuthorizeAudio confere que a chamada existe NESTA empresa, que está em curso e
// que o operador é quem a assumiu. Qualquer outra pessoa recebe ErrNotFound (nunca
// revela que a chamada existe) ou ErrNotYourCall.
func (s *Service) AuthorizeAudio(tenantID uuid.UUID, userID int, callID string) error {
	l, err := s.load(tenantID, callID)
	if err != nil {
		return ErrNotFound
	}
	if l.EndedAt != nil {
		return ErrNotActive
	}
	if l.HandledByUserID == nil || *l.HandledByUserID != userID {
		return ErrNotYourCall
	}
	return nil
}

// ServeAudio liga o WebSocket do navegador (já aceito) ao do engine até um dos
// lados cair ou a chamada acabar. Quando o navegador cai, arma o watchdog: sem
// voltar em DropGrace a chamada é encerrada (o engine desconecta o contato).
func (s *Service) ServeAudio(ctx context.Context, a *Audio, dial EngineDialer, browser *websocket.Conn, tenantID uuid.UUID, userID int, callID string) {
	defer func() { _ = browser.CloseNow() }()
	br, err := a.Open(callID)
	if err != nil {
		_ = browser.Close(websocket.StatusPolicyViolation, "canal de áudio já aberto")
		return
	}
	defer a.Release(br)

	// A gravação lê o MESMO áudio que atravessa o proxy, nos dois sentidos. Aqui só se liga o
	// Tap; quem INICIA a gravação é o call.state "active" (modo automático, HandleState) ou o
	// pedido do operador (modo opcional). Assim o gatilho é um só, nos dois sentidos, o mesmo que
	// liga o cronômetro. Feed é no-op sem gravação ativa, e uma já em curso (o operador
	// reconectou o áudio) continua.
	if s.rec != nil {
		br.Tap = func(fromOperator bool, frame []byte) { s.rec.Feed(tenantID, callID, fromOperator, frame) }
	}

	eng, err := dial(ctx, callID)
	if err != nil {
		slog.Error("canal de áudio do engine indisponível: a chamada será encerrada",
			"callId", callID, "tenantId", tenantID, "err", err)
		_ = browser.Close(websocket.StatusInternalError, "audio_unavailable")
		_ = s.End(ctx, tenantID, userID, callID)
		return
	}
	defer func() { _ = eng.CloseNow() }()
	eng.SetReadLimit(1 << 20)
	browser.SetReadLimit(maxVideoMessage)

	rctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() { // navegador → engine
		defer cancel()
		for {
			typ, data, err := browser.Read(rctx)
			if err != nil {
				return
			}
			switch {
			case typ == websocket.MessageBinary && len(data) > 0:
				br.FromBrowser(data)
			case typ == websocket.MessageText:
				if c, ok := parseCameraCommand(data); ok {
					raw, _ := json.Marshal(c)
					wctx, wc := context.WithTimeout(rctx, 2*time.Second)
					_ = eng.Write(wctx, websocket.MessageText, raw)
					wc()
				}
			}
		}
	}()
	go func() { // engine → navegador (áudio) e → business (telemetria)
		defer cancel()
		for {
			typ, data, err := eng.Read(rctx)
			if err != nil {
				return
			}
			switch typ {
			case websocket.MessageBinary:
				br.FromEngine(data)
			case websocket.MessageText:
				if isKeyframeRequest(data) {
					wctx, wc := context.WithTimeout(rctx, 2*time.Second)
					_ = browser.Write(wctx, websocket.MessageText, data)
					wc()
					continue
				}
				s.handleTelemetry(ctx, tenantID, data)
			}
		}
	}()
	go pump(rctx, cancel, br.ToEngine, eng)
	go pump(rctx, cancel, br.VideoToEngine, eng)
	go pump(rctx, cancel, br.ToBrowser, browser)

	<-rctx.Done()
	if !a.Active(callID) {
		return
	}
	a.Release(br)
	ended := func() bool {
		l, err := s.load(tenantID, callID)
		return err != nil || l.EndedAt != nil
	}
	go a.Watchdog(context.Background(), callID, ended, func() {
		_ = s.End(context.Background(), tenantID, userID, callID)
	})
}

func pump(ctx context.Context, cancel context.CancelFunc, from *Pipe, to *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-from.Out():
			wctx, c := context.WithTimeout(ctx, 2*time.Second)
			err := to.Write(wctx, websocket.MessageBinary, f)
			c()
			if err != nil {
				cancel()
				return
			}
		}
	}
}

// handleTelemetry repassa a medição do engine ao tratador de qualidade, que a
// interpreta e entrega só ao operador da chamada.
func (s *Service) handleTelemetry(ctx context.Context, tenantID uuid.UUID, raw []byte) {
	var t Telemetry
	if json.Unmarshal(raw, &t) != nil || t.Type != "quality" || t.CallID == "" {
		return
	}
	_ = s.HandleQuality(ctx, tenantID, raw)
}

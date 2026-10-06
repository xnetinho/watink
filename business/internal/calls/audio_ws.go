package calls

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// EngineDialer abre o WebSocket interno de áudio do engine para uma chamada.
type EngineDialer func(ctx context.Context, callID string) (*websocket.Conn, error)

// NewEngineDialer devolve o discador de produção: base é "ws://engine:8085" e
// token o CALLS_AUDIO_TOKEN compartilhado. Sem base ou token não há áudio.
func NewEngineDialer(base, token string) EngineDialer {
	return func(ctx context.Context, callID string) (*websocket.Conn, error) {
		if base == "" || token == "" {
			return nil, errors.New("áudio de chamadas não configurado (CALLS_AUDIO_URL/CALLS_AUDIO_TOKEN)")
		}
		hdr := http.Header{}
		hdr.Set("X-Internal-Token", token)
		c, _, err := websocket.Dial(ctx, base+"/calls/"+callID+"/audio", &websocket.DialOptions{HTTPHeader: hdr})
		return c, err
	}
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
	defer browser.CloseNow()
	br, err := a.Open(callID)
	if err != nil {
		_ = browser.Close(websocket.StatusPolicyViolation, "canal de áudio já aberto")
		return
	}
	defer a.Release(br)

	// A gravação lê o MESMO áudio que atravessa o proxy, nos dois sentidos. No modo
	// automático ela começa aqui, ao conectar o áudio; no opcional, só quando o
	// operador pede. Chamadas já em gravação (ex.: o operador reconectou) seguem.
	if s.rec != nil {
		br.Tap = func(fromOperator bool, frame []byte) { s.rec.Feed(tenantID, callID, fromOperator, frame) }
		if s.autoRecord(tenantID) {
			s.startRecordingBestEffort(tenantID, userID, callID)
		}
	}

	eng, err := dial(ctx, callID)
	if err != nil {
		_ = browser.Close(websocket.StatusInternalError, "áudio indisponível")
		_ = s.End(ctx, tenantID, userID, callID)
		return
	}
	defer eng.CloseNow()
	eng.SetReadLimit(1 << 20)
	browser.SetReadLimit(64 * 1024)

	rctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() { // navegador → engine
		defer cancel()
		for {
			typ, data, err := browser.Read(rctx)
			if err != nil {
				return
			}
			if typ == websocket.MessageBinary && len(data) > 0 {
				br.FromBrowser(data)
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
				s.handleTelemetry(ctx, tenantID, data)
			}
		}
	}()
	go pump(rctx, cancel, br.ToEngine, eng)
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

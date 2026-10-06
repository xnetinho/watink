package calls

import (
	"context"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

const (
	// readyTimeout: tempo que o business tem para confirmar operador elegível
	// (comando call.ready) antes de a oferta ser abandonada.
	readyTimeout = 3 * time.Second
	// ringTimeout: toque máximo, para chamadas recebidas e originadas.
	ringTimeout = 45 * time.Second
)

// Motivos de chamada não atendida/encerrada que o engine publica.
const (
	ReasonBusy            = "busy"
	ReasonProxyBlocked    = "proxy_blocked"
	ReasonUnsupportedType = "unsupported_type"
	ReasonNoOperator      = "no_operator"
	ReasonTimeout         = "timeout"
	ReasonCancelled       = "cancelled"
	ReasonAcceptedElse    = "accepted_elsewhere"
	ReasonInterrupted     = "interrupted"
	ReasonFailed          = "failed"
)

// PublishFunc entrega um evento ao broker (wbot.<tenant>.<session>.<type>).
type PublishFunc func(tenantID string, sessionID int, eventType string, payload map[string]interface{})

// State é o recorte do estado de uma chamada que o engine propaga.
type State struct {
	CallID       string
	State        string
	Direction    string
	EndReason    string
	DurationSecs int
}

// Hooks são os callbacks que o gerenciador de chamada dispara. ATENÇÃO: o
// CallManager chama OnState com o próprio mutex travado, então um hook nunca
// pode chamar de volta o Handle de forma síncrona.
type Hooks struct {
	OnState func(State)
	OnEnded func(State)
}

// Handle é o que o engine usa de um gerenciador de chamada. A implementação real
// envolve *call.CallManager; os testes usam um fake.
type Handle interface {
	SetHooks(Hooks)
	HandleOffer(ctx context.Context, node *waBinary.Node, from types.JID)
	HandleAccept(ctx context.Context, node *waBinary.Node, from types.JID)
	HandleTransport(ctx context.Context, node *waBinary.Node, from types.JID)
	HandleTerminate(node *waBinary.Node)
	HandleRelayLatency(node *waBinary.Node)
	SendPreaccept(ctx context.Context) error
	Accept(ctx context.Context, callID string) error
	Reject(ctx context.Context, callID string) error
	End(ctx context.Context, reason string) error
	Start(ctx context.Context, callID string, peer types.JID) error
	Abandon(reason string)
}

// NewHandleFunc cria o gerenciador de uma chamada sobre o socket da sessão.
type NewHandleFunc func(sock core.VoipSocket) Handle

type stopper interface{ Stop() bool }

// clock existe para os testes controlarem os temporizadores (3 s / 45 s).
type clock interface {
	AfterFunc(d time.Duration, f func()) stopper
}

type realClock struct{}

func (realClock) AfterFunc(d time.Duration, f func()) stopper { return time.AfterFunc(d, f) }

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
	// mediaConnectTimeout: depois de atendida, a mídia (relay UDP) precisa conectar
	// neste prazo. Sem ele, uma saída UDP bloqueada deixaria a chamada em
	// "Conectando…" para sempre, ocupando a conexão. O relay já desiste sozinho aos
	// 20 s; este prazo é o que encerra a CHAMADA.
	mediaConnectTimeout = 25 * time.Second
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
	// ReasonMediaTimeout: a chamada foi atendida mas o áudio nunca conectou
	// (tipicamente saída UDP do servidor bloqueada).
	ReasonMediaTimeout = "media_timeout"
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

	SetMedia(MediaHooks)
	FeedPCM(pcm []float32)
	// SendVideo envia uma access unit H.264 (Annex-B) da câmera do operador; duration é o tempo que o
	// quadro representa (0 = o padrão de 15 fps). Nunca bloqueia.
	SendVideo(accessUnit []byte, duration time.Duration)
	// SetCamera liga ou desliga a câmera do operador: avisa o contato e passa a aceitar ou descartar
	// quadros. orientation vai de 0 a 3 (quartos de volta horários).
	SetCamera(ctx context.Context, on bool, orientation int) error
	RelayRTTMs() (int, bool)
	RelayConnected() bool
}

// MediaHooks são os ganchos de mídia de uma chamada, ligados ao canal de áudio.
type MediaHooks struct {
	OnPeerPCM func(pcm []float32)
	OnPeerRtp func(seq uint16, timestamp uint32, payloadLen int)
	OnSentRtp func(size int)
	// OnPeerVideo recebe cada access unit H.264 (Annex-B) completa do contato. Roda na goroutine do
	// relay: nunca pode bloquear.
	OnPeerVideo func(accessUnit []byte, keyframe bool)
	// OnPeerVideoFrame é como OnPeerVideo, com a rotação (0..3) anunciada pelo aparelho do contato.
	OnPeerVideoFrame func(accessUnit []byte, keyframe bool, rotation int)
	// OnKeyframeRequested avisa que o contato pediu um quadro-chave do vídeo do operador: o codificador
	// da câmera precisa gerar um IDR agora. Roda na goroutine do relay: nunca pode bloquear.
	OnKeyframeRequested func()
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

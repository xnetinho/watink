package calls

import (
	"context"
	"log/slog"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/call"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

type managerHandle struct{ m *call.CallManager }

// NewManagerHandle é o NewHandleFunc de produção.
func NewManagerHandle(sock core.VoipSocket) Handle {
	m := call.NewCallManager(sock, slog.Default())
	m.DeferPreaccept = true
	return &managerHandle{m: m}
}

func stateOf(c *call.CallInfo) State {
	return State{
		CallID:       c.CallID,
		State:        string(c.StateData.State),
		Direction:    string(c.Direction),
		EndReason:    string(c.StateData.EndReason),
		DurationSecs: c.StateData.DurationSecs,
	}
}

func (h *managerHandle) SetHooks(k Hooks) {
	h.m.OnStateChange = func(c *call.CallInfo) {
		if k.OnState != nil {
			k.OnState(stateOf(c))
		}
	}
	h.m.OnEnded = func(c *call.CallInfo) {
		if k.OnEnded != nil {
			k.OnEnded(stateOf(c))
		}
	}
}

func (h *managerHandle) HandleOffer(ctx context.Context, n *waBinary.Node, from types.JID) {
	h.m.HandleCallOffer(ctx, n, from)
}
func (h *managerHandle) HandleAccept(ctx context.Context, n *waBinary.Node, from types.JID) {
	h.m.HandleCallAccept(ctx, n, from)
}
func (h *managerHandle) HandleTransport(ctx context.Context, n *waBinary.Node, from types.JID) {
	h.m.HandleCallTransport(ctx, n, from)
}
func (h *managerHandle) HandleTerminate(n *waBinary.Node) { h.m.HandleCallTerminate(n) }

// HandleRelayLatency recebe a medição de latência do relay; o uso dela (RTT da
// telemetria) é implementado junto da medição de qualidade.
func (h *managerHandle) HandleRelayLatency(*waBinary.Node)       {}
func (h *managerHandle) SendPreaccept(ctx context.Context) error { return h.m.SendPreaccept(ctx) }
func (h *managerHandle) Accept(ctx context.Context, callID string) error {
	return h.m.AcceptCall(ctx, callID)
}
func (h *managerHandle) Reject(ctx context.Context, callID string) error {
	return h.m.RejectCall(ctx, callID, core.EndCallReasonDeclined)
}
func (h *managerHandle) End(ctx context.Context, reason string) error {
	return h.m.EndCall(ctx, core.EndCallReason(reason))
}
func (h *managerHandle) Start(ctx context.Context, callID string, peer types.JID) error {
	return h.m.StartCall(ctx, callID, peer, false)
}
func (h *managerHandle) SetMedia(k MediaHooks) {
	h.m.OnPeerAudio = k.OnPeerPCM
	h.m.OnPeerRtp = k.OnPeerRtp
	h.m.OnSentRtp = k.OnSentRtp
	h.m.OnPeerVideo = k.OnPeerVideo
	h.m.OnPeerVideoFrame = k.OnPeerVideoFrame
}
func (h *managerHandle) FeedPCM(pcm []float32)   { h.m.FeedCapturedPCM(pcm) }
func (h *managerHandle) RelayRTTMs() (int, bool) { return h.m.RelayRTTMs() }
func (h *managerHandle) RelayConnected() bool    { return h.m.RelayConnected() }
func (h *managerHandle) Abandon(reason string)   { h.m.AbandonCall(core.EndCallReason(reason)) }

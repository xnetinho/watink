package call

import (
	"context"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/signaling"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/wanode"
)

// Este arquivo NÃO vem do WaCalls: reúne as duas adições do Watink (ver
// NOTICE.md). Ficam à parte para que atualizar o porte não as sobrescreva.

// SendPreaccept envia o `preaccept` adiado por DeferPreaccept. Só vale para uma
// chamada recebida ainda tocando; qualquer outro estado é um no-op.
func (m *CallManager) SendPreaccept(ctx context.Context) error {
	m.mu.Lock()
	call := m.currentCall
	if call == nil || call.IsEnded() || call.Direction != core.CallDirectionIncoming || !call.CanAccept() {
		m.mu.Unlock()
		return nil
	}
	node := signaling.BuildPreacceptStanza(wanode.MustJID(call.PeerJid), call.CallID, wanode.MustJID(call.CallCreator), call.MediaType == core.CallMediaTypeVideo)
	m.mu.Unlock()
	return m.sock.SendNode(ctx, node)
}

// AbandonCall encerra a chamada só localmente: nenhum stanza (nem reject, nem
// terminate) é enviado. É o caminho de uma oferta ignorada, ou atendida em
// outro aparelho, em que mandar `reject` derrubaria o toque dos demais
// aparelhos da conta. Libera a mídia e dispara OnEnded.
func (m *CallManager) AbandonCall(reason core.EndCallReason) {
	m.mu.Lock()
	call := m.currentCall
	if call == nil || call.IsEnded() {
		m.mu.Unlock()
		return
	}
	_ = call.ApplyTransition(Transition{Type: TransitionTerminated, Reason: reason})
	m.emitState()
	m.mu.Unlock()

	if m.OnEnded != nil {
		m.OnEnded(call)
	}
	m.cleanupMedia()
}

// RelayRTTMs devolve o RTT do relay em ms, vindo do `c2r_rtt` da oferta (o menor
// entre os relays). ok=false se a oferta não trouxe a medida.
func (m *CallManager) RelayRTTMs() (rtt int, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentCall == nil || m.currentCall.RelayData == nil {
		return 0, false
	}
	for _, ep := range m.currentCall.RelayData.Endpoints {
		if ep.C2RRtt != nil && (!ok || *ep.C2RRtt < rtt) {
			rtt, ok = *ep.C2RRtt, true
		}
	}
	return rtt, ok
}

// RelayConnected diz se há ao menos um relay com a mídia aberta.
func (m *CallManager) RelayConnected() bool { return m.relay != nil && m.relay.HasConnection() }

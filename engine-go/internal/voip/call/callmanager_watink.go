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
	node := signaling.BuildPreacceptStanza(wanode.MustJID(call.PeerJid), call.CallID, wanode.MustJID(call.CallCreator))
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

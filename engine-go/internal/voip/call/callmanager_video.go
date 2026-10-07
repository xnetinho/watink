package call

import (
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/media"
)

// SSRCs de vídeo: o mesmo HKDF do áudio (callID, JID do aparelho), com o slot 2 no lugar do 0. O
// vídeo é uma mídia à parte da mesma chamada, com SSRC, sequência e ROC próprios, mas as mesmas chaves.

// isVideoCallLocked diz se a chamada atual foi negociada com vídeo. Precisa estar com m.mu travado.
func (m *CallManager) isVideoCallLocked() bool {
	return m.currentCall != nil && m.currentCall.MediaType == core.CallMediaTypeVideo
}

// deriveVideoSsrcsLocked calcula os SSRCs de vídeo a partir dos JIDs de aparelho já conhecidos para o
// áudio. Chamar de novo depois de refinar o JID do contato (ack do relay, accept) recalcula. Sem
// vídeo na chamada não faz nada. Precisa estar com m.mu travado.
func (m *CallManager) deriveVideoSsrcsLocked(selfDeviceJid, peerDeviceJid string) {
	if !m.isVideoCallLocked() {
		return
	}
	callID := m.currentCall.CallID
	if selfDeviceJid != "" {
		m.videoSelfSsrc = media.GenerateSecureSsrc(callID, selfDeviceJid, media.VideoSlotWord)
	}
	if peerDeviceJid != "" {
		m.videoPeerSsrc = media.GenerateSecureSsrc(callID, peerDeviceJid, media.VideoSlotWord)
	}
}

// applyVideoSsrcsLocked entrega ao relay os SSRCs de vídeo, para a alocação incluí-los. Precisa estar
// com m.mu travado.
func (m *CallManager) applyVideoSsrcsLocked() {
	var self, peer []uint32
	if m.videoSelfSsrc != 0 {
		self = append(self, m.videoSelfSsrc)
	}
	if m.videoPeerSsrc != 0 {
		peer = append(peer, m.videoPeerSsrc)
	}
	m.relay.SetExtraSsrcs(self, peer)
}

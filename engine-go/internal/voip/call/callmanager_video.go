package call

import (
	"time"

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

// videoPLIInterval limita os pedidos de quadro-chave: uma rajada de perda não pode inundar o contato.
const videoPLIInterval = 300 * time.Millisecond

// onVideoRtp trata um pacote RTP de vídeo (PT 97) vindo do relay: autentica e decifra com o contexto
// SRTP do SSRC de vídeo, remonta a access unit e a entrega. Em lacuna de sequência descarta o quadro
// incompleto, pede um quadro-chave (com intervalo mínimo) e só volta a entregar a partir de um IDR.
// Não toca na subscrição de áudio (peerSsrcs/actualPeerSet): vídeo e áudio têm fluxos separados.
func (m *CallManager) onVideoRtp(data []byte, ssrc uint32) {
	m.mu.Lock()
	if m.srtpSession == nil || !m.isVideoCallLocked() || ssrc == m.videoSelfSsrc {
		m.mu.Unlock()
		return
	}
	srtp := m.srtpSession
	m.mu.Unlock()

	pkt, err := srtp.Unprotect(data)
	if err != nil {
		m.log.Debug("srtp video unprotect error", "err", err)
		return
	}
	if len(pkt.Payload) == 0 || pkt.Header == nil {
		return
	}

	// CVO: os 2 bits baixos do MediaFrameInfo dizem quantos quartos de volta (horários) o receptor deve
	// girar. Sem a extensão, a imagem fica como veio (0).
	rotation, hasRotation := 0, false
	if ext, ok := media.ParseVideoRtpExtension(pkt.Header); ok {
		rotation, hasRotation = ext.DisplayOrientation(), true
	}

	m.mu.Lock()
	rotationChanged := false
	if hasRotation && m.videoOrientation != rotation {
		m.videoOrientation = rotation
		rotationChanged = true
	}
	au, ok, recovery := m.videoRx.Push(pkt.Header.SequenceNumber, pkt.Header.Marker, pkt.Payload)
	needPLI := false
	if recovery && time.Since(m.videoLastPLI) >= videoPLIInterval {
		m.videoLastPLI = time.Now()
		needPLI = true
	}
	onVideo, onVideoRot, onPLI := m.OnPeerVideo, m.OnPeerVideoFrame, m.OnVideoKeyframeNeeded
	onOrient := m.OnPeerVideoOrientation
	current := m.videoOrientation
	if current < 0 {
		current = 0
	}
	m.mu.Unlock()

	if needPLI && onPLI != nil {
		onPLI()
	}
	if rotationChanged && onOrient != nil {
		onOrient(rotation)
	}
	if ok {
		key := media.AUHasIDR(au)
		if onVideoRot != nil {
			onVideoRot(au, key, current)
		}
		if onVideo != nil {
			onVideo(au, key)
		}
	}
}

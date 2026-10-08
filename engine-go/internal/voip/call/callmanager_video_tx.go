package call

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/media"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/signaling"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/wanode"
)

// Envio de vídeo (fase 2 do plano add-whatsapp-video-calls): a câmera do operador chega como access units
// H.264 Annex-B já codificadas pelo navegador (WebCodecs); aqui só se empacota, cifra e envia. O engine não
// codifica pixel nenhum.
//
// NÃO VALIDADO AO VIVO: o caminho de envio de vídeo é o único que nenhuma das bases de referência provou.
// Por isso cada passo deixa um log (primeiro quadro, primeiro RTCP autenticado do contato, pedido de IDR).

const (
	// videoSRInterval é o intervalo do Sender Report + SDES do vídeo (o da meowcaller). Sem ele o vídeo do
	// lado que envia não passa a fluir para o outro.
	videoSRInterval = 1500 * time.Millisecond
	// videoDefaultStride é o avanço do relógio de 90 kHz quando não se sabe a duração do quadro (15 fps).
	videoDefaultStride = 6000
)

var errNoVideoCall = errors.New("a chamada não tem vídeo")

// deriveSrtcpKeysLocked deriva as chaves de SRTCP das mesmas chaves-mestras por JID do SRTP. Precisa estar
// com m.mu travado.
func (m *CallManager) deriveSrtcpKeysLocked(sendKM, recvKM core.SrtpKeyingMaterial) {
	send, err1 := media.DeriveSrtcpKeys(sendKM)
	recv, err2 := media.DeriveSrtcpKeys(recvKM)
	if err1 != nil || err2 != nil {
		m.log.Error("srtcp key derivation failed", "err1", err1, "err2", err2)
		return
	}
	m.srtcpSend, m.srtcpRecv = send, recv
}

// resetVideoTxLocked zera o estado de envio de vídeo e para o laço do Sender Report. Precisa estar com
// m.mu travado.
func (m *CallManager) resetVideoTxLocked() {
	if m.rtcpStop != nil {
		close(m.rtcpStop)
		m.rtcpStop = nil
	}
	m.videoTx, m.videoTxSsrc = nil, 0
	m.videoTxOn, m.videoNeedIDR = false, false
	m.videoPackets, m.videoOctets, m.videoLastTs = 0, 0, 0
	m.srtcpSend, m.srtcpRecv, m.srtcpIndex = nil, nil, 0
	m.rtcpInSeen, m.rtcpAuthFail = false, false
}

func videoStride(d time.Duration) uint32 {
	if d <= 0 {
		return videoDefaultStride
	}
	if ticks := uint32((d.Nanoseconds()*media.VideoClockRate + int64(time.Second)/2) / int64(time.Second)); ticks > 0 {
		return ticks
	}
	return videoDefaultStride
}

// SendVideoFrame envia uma access unit H.264 (Annex-B) da câmera do operador. Descarta em silêncio o que
// não pode ser enviado agora: câmera desligada, chamada sem vídeo, mídia ainda não pronta, e deltas
// enquanto o contato não tiver um IDR (primeiro quadro e depois de cada PLI). duration é o tempo que o
// quadro representa (0 = 15 fps).
func (m *CallManager) SendVideoFrame(au []byte, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.videoTxOn || !m.isVideoCallLocked() || m.srtpSession == nil || m.srtcpSend == nil ||
		m.videoSelfSsrc == 0 || !m.relay.HasConnection() {
		return
	}
	idr := media.AUHasIDR(au)
	if m.videoNeedIDR && !idr {
		return
	}
	packed := media.PackAccessUnit(au)
	if len(packed) == 0 {
		return
	}
	if m.videoTx == nil || m.videoTxSsrc != m.videoSelfSsrc {
		m.videoTx, m.videoTxSsrc = media.NewVideoRtpStream(m.videoSelfSsrc, videoDefaultStride), m.videoSelfSsrc
	}
	m.videoTx.SetTimestampStride(videoStride(duration))
	info := media.VideoFrameInfoDelta
	if idr {
		info = media.VideoFrameInfoIDR
	}
	payloads := media.PackageH264NALU(packed)
	sent := 0
	for i, p := range payloads {
		h, _ := m.videoTx.NextPacket(i == len(payloads)-1, info)
		prot, err := m.srtpSession.Protect(&media.RtpPacket{Header: h, Payload: p})
		if err != nil {
			m.log.Debug("srtp video protect error", "err", err)
			continue
		}
		m.relay.Broadcast(prot)
		m.videoPackets++
		m.videoOctets += uint32(len(p))
		m.videoLastTs = h.Timestamp
		sent++
	}
	if sent == 0 {
		return
	}
	if m.videoPackets == uint32(sent) {
		m.log.Info("first video frame sent", "bytes", len(au), "idr", idr, "packets", sent, "ssrc", m.videoSelfSsrc)
		m.startVideoRtcpLocked()
	}
	if idr {
		m.videoNeedIDR = false
	}
}

// startVideoRtcpLocked liga o laço do Sender Report + SDES; é chamado no primeiro quadro enviado.
func (m *CallManager) startVideoRtcpLocked() {
	if m.rtcpStop != nil {
		return
	}
	var entropy [12]byte
	_, _ = rand.Read(entropy[:])
	m.rtcpCname = media.BuildRtcpCname(entropy)
	stop := make(chan struct{})
	m.rtcpStop = stop
	go func() {
		t := time.NewTicker(videoSRInterval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case now := <-t.C:
				m.sendVideoSenderReport(now)
			}
		}
	}()
}

func (m *CallManager) sendVideoSenderReport(now time.Time) {
	m.mu.Lock()
	if m.srtcpSend == nil || m.videoPackets == 0 || m.videoSelfSsrc == 0 || !m.relay.HasConnection() {
		m.mu.Unlock()
		return
	}
	stats := media.RtcpSenderStats{PacketsSent: m.videoPackets, OctetsSent: m.videoOctets, RtpTimestamp: m.videoLastTs}
	plain := media.BuildSenderReportWithSdes(m.videoSelfSsrc, stats, uint64(now.UnixMilli()), &m.rtcpCname, true)
	m.srtcpIndex++
	pkt, err := m.srtcpSend.Protect(m.videoSelfSsrc, m.srtcpIndex, plain)
	m.mu.Unlock()
	if err != nil {
		m.log.Debug("srtcp protect error", "err", err)
		return
	}
	m.relay.Broadcast(pkt)
}

// sendVideoPLI pede ao contato um quadro-chave do vídeo dele (perdemos um pacote).
func (m *CallManager) sendVideoPLI(peerVideoSsrc uint32) {
	m.mu.Lock()
	if m.srtcpSend == nil || m.videoSelfSsrc == 0 || peerVideoSsrc == 0 || !m.relay.HasConnection() {
		m.mu.Unlock()
		return
	}
	plain := media.BuildPictureLossIndication(m.videoSelfSsrc, peerVideoSsrc, true)
	m.srtcpIndex++
	pkt, err := m.srtcpSend.Protect(m.videoSelfSsrc, m.srtcpIndex, plain[:])
	m.mu.Unlock()
	if err != nil {
		m.log.Debug("srtcp pli protect error", "err", err)
		return
	}
	m.relay.Broadcast(pkt)
	m.log.Debug("video PLI sent", "media_ssrc", peerVideoSsrc)
}

// onRtcp trata um pacote RTCP do contato: autentica e decifra (SRTCP) e, se ele pede um quadro-chave do
// nosso vídeo, marca que só um IDR pode sair a seguir e avisa o codificador.
func (m *CallManager) onRtcp(data []byte) {
	if len(data) < 8 {
		return
	}
	sender := binary.BigEndian.Uint32(data[4:8])
	m.mu.Lock()
	keys, own, cb := m.srtcpRecv, m.videoSelfSsrc, m.OnVideoKeyframeRequested
	m.mu.Unlock()
	if keys == nil {
		return
	}
	plain, index, err := keys.Unprotect(sender, data)
	if err != nil {
		m.mu.Lock()
		first := !m.rtcpAuthFail
		m.rtcpAuthFail = true
		m.mu.Unlock()
		if first {
			m.log.Warn("peer SRTCP failed authentication", "ssrc", sender, "bytes", len(data))
		}
		return
	}
	m.mu.Lock()
	first := !m.rtcpInSeen
	m.rtcpInSeen = true
	m.mu.Unlock()
	if first {
		m.log.Info("first authenticated peer SRTCP received", "ssrc", sender, "index", index)
	}
	if own == 0 || !media.RtcpRequestsKeyframe(plain, own) {
		return
	}
	m.mu.Lock()
	m.videoNeedIDR = true
	m.mu.Unlock()
	m.log.Debug("peer requested a video keyframe", "video_ssrc", own)
	if cb != nil {
		cb()
	}
}

// SetLocalVideo liga ou desliga a câmera do operador na chamada: avisa o contato (estado 1 com a
// orientação do aparelho, ou 6) e passa a aceitar ou descartar quadros. Ligar exige que o primeiro quadro
// enviado seja um IDR. orientation vai de 0 a 3 (quartos de volta horários).
func (m *CallManager) SetLocalVideo(ctx context.Context, on bool, orientation int) error {
	m.mu.Lock()
	call := m.currentCall
	if call == nil || call.IsEnded() || !m.isVideoCallLocked() {
		m.mu.Unlock()
		return errNoVideoCall
	}
	params := signaling.VideoStateParams{
		CallID: call.CallID, To: wanode.MustJID(call.PeerJid), CallCreator: wanode.MustJID(call.CallCreator),
		WrapperID: signaling.GenerateCallStanzaID(), State: signaling.VideoStateStopped,
	}
	if on {
		params.State = signaling.VideoStateEnabled
		if orientation < 0 || orientation > 3 {
			orientation = 0
		}
		params.DeviceOrientation = &orientation
	}
	m.videoTxOn = on
	if on {
		m.videoNeedIDR = true
	}
	m.mu.Unlock()
	return m.sock.SendNode(ctx, signaling.BuildVideoStateWithParams(params))
}

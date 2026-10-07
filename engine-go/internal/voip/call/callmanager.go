package call

import (
	"context"
	"log/slog"
	"sync"
	"time"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/media"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/signaling"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/transport"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/wanode"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

type CallManager struct {
	sock core.VoipSocket
	log  *slog.Logger

	mu          sync.Mutex
	currentCall *CallInfo

	rtpSession  *media.RtpSession
	srtpSession *media.SrtpSession
	codec       media.Codec
	relay       RelayTransport

	selfSsrc      uint32
	peerSsrcs     []uint32
	actualPeerSet bool

	// Vídeo (fases 0/1 do plano add-whatsapp-video-calls). O contato fala em um SSRC próprio, derivado
	// com o slot 2 do mesmo HKDF do áudio; sem ele na alocação do relay o vídeo dele não chega.
	videoSelfSsrc uint32
	videoPeerSsrc uint32

	firstPacketSent       bool
	initialTransportSent  bool
	outgoingPreacceptSent bool
	acceptedByJid         string
	debeEnabled           bool

	encodeBuf    []float32
	encodeBufPos int

	lastCaptureAt time.Time
	keepaliveStop chan struct{}

	// DeferPreaccept (alteração Watink): em vez de enviar `preaccept` ao receber
	// a oferta, espera SendPreaccept. O engine só sinaliza "estou atendendo" ao
	// chamador depois que o business confirma que há operador elegível.
	DeferPreaccept bool

	OnStateChange func(*CallInfo)
	OnIncoming    func(*CallInfo)
	OnEnded       func(*CallInfo)
	OnPeerAudio   func([]float32)

	// OnPeerRtp (alteração Watink) recebe sequência, timestamp e tamanho do payload
	// de cada pacote RTP do contato já autenticado, para medir perda, jitter e taxa.
	OnPeerRtp func(seq uint16, timestamp uint32, payloadLen int)
	// OnSentRtp (alteração Watink) recebe o tamanho de cada pacote enviado ao relay.
	OnSentRtp func(size int)
}

func NewCallManager(sock core.VoipSocket, log *slog.Logger) *CallManager {
	if log == nil {
		log = slog.Default()
	}
	m := &CallManager{
		sock:        sock,
		log:         log,
		debeEnabled: true,
	}
	relay := transport.NewSctpRelayManager(log)
	relay.SetOnConnected(func(ip string, port int) { m.onRelayConnected() })
	relay.SetOnReceive(func(data []byte) { m.onRelayData(data) })
	m.relay = relay
	return m
}

func (m *CallManager) CurrentCall() *CallInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentCall
}

func (m *CallManager) emitState() {
	if m.OnStateChange != nil && m.currentCall != nil {
		m.OnStateChange(m.currentCall)
	}
}

func (m *CallManager) StartCall(ctx context.Context, callID string, peerJid types.JID, isVideo bool) error {
	m.mu.Lock()
	if m.currentCall != nil && !m.currentCall.IsEnded() {
		m.mu.Unlock()
		return &CallError{"a call is already in progress"}
	}

	mediaType := core.CallMediaTypeAudio
	if isVideo {
		mediaType = core.CallMediaTypeVideo
	}
	creator := m.sock.OwnLID()
	if creator.IsEmpty() {
		creator = m.sock.OwnPN()
	}
	resolved := m.sock.ResolveLIDForPN(ctx, peerJid)

	call := NewOutgoingCall(callID, resolved.String(), creator.String(), mediaType)
	callKey := media.GenerateCallKey()
	call.EncryptionKey = callKey
	m.currentCall = call
	m.initialTransportSent = false
	m.outgoingPreacceptSent = false

	selfJid := creator.String()
	m.selfSsrc = media.GenerateSecureSsrc(callID, selfJid, 0)
	m.rtpSession = media.NewWhatsAppOpusSession(m.selfSsrc)
	m.peerSsrcs = []uint32{media.GenerateSecureSsrc(callID, resolved.String(), 0)}
	m.deriveVideoSsrcsLocked(selfJid, resolved.String())
	m.applyVideoSsrcsLocked()
	m.initCodec()
	m.mu.Unlock()

	offer, err := signaling.BuildOfferStanza(ctx, m.sock, callID, callKey, resolved, isVideo)
	if err != nil {
		return err
	}
	ackNode, err := m.sock.Query(ctx, offer)
	if err != nil {
		return err
	}

	m.mu.Lock()
	_ = m.currentCall.ApplyTransition(Transition{Type: TransitionOfferSent})
	m.emitState()
	m.mu.Unlock()

	if ackNode != nil {
		go m.HandleCallAck(context.Background(), ackNode)
	}

	m.log.Info("call offer sent", "call_id", callID, "peer", resolved.String())
	return nil
}

func (m *CallManager) AcceptCall(ctx context.Context, callID string) error {
	m.mu.Lock()
	call := m.currentCall
	if call == nil || call.CallID != callID {
		m.mu.Unlock()
		return &CallError{"no incoming call with id " + callID}
	}
	if !call.CanAccept() {
		m.mu.Unlock()
		return &CallError{"call cannot be accepted in state " + string(call.StateData.State)}
	}
	_ = call.ApplyTransition(Transition{Type: TransitionLocalAccepted})
	m.emitState()
	key := call.EncryptionKey
	peer := wanode.MustJID(call.PeerJid)
	creator := wanode.MustJID(call.CallCreator)
	isVideo := call.MediaType == core.CallMediaTypeVideo
	relayData := call.RelayData
	m.mu.Unlock()

	if key != nil {
		acceptNode, err := signaling.BuildAcceptStanza(ctx, m.sock, callID, key, peer, creator, isVideo)
		if err != nil {
			m.log.Error("build accept failed", "err", err)
		} else if err := m.sock.SendNode(ctx, acceptNode); err != nil {
			m.log.Error("accept send error", "err", err)
		}
	}

	if relayData != nil {
		m.setupIncomingMedia(call, relayData)
		m.connectRelays(relayData.Endpoints)
	} else {
		m.log.Warn("call accepted but no relay endpoints yet; media path waits for a transport message", "call_id", callID)
	}
	m.log.Info("call accepted", "call_id", callID)
	return nil
}

func (m *CallManager) setupIncomingMedia(call *CallInfo, relayData *core.RelayData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(relayData.ParticipantJids) > 0 {
		ourBase := wanode.CleanJID(m.ownCredJid())
		ourDeviceJid := ensureDeviceJid(findOurDevice(relayData.ParticipantJids, ourBase, m.ownCredJid()))
		if newSelf := media.GenerateSecureSsrc(call.CallID, ourDeviceJid, 0); newSelf != m.selfSsrc {
			m.selfSsrc = newSelf
			m.rtpSession = media.NewWhatsAppOpusSession(newSelf)
		}
		peerDevice := ""
		if peer := firstPeerDevice(relayData.ParticipantJids, ourBase); peer != "" {
			peerDevice = ensureDeviceJid(peer)
			m.peerSsrcs = []uint32{media.GenerateSecureSsrc(call.CallID, peerDevice, 0)}
			m.actualPeerSet = true
		}
		m.deriveVideoSsrcsLocked(ourDeviceJid, peerDevice)
	}
	m.applyVideoSsrcsLocked()
	m.relay.SetSubscriptionSsrc(firstSsrc(m.peerSsrcs))
	m.initSrtpKeysLocked()
}

const signalingSendTimeout = 10 * time.Second

// sendSignaling envia reject/terminate sem depender do contexto de quem pediu: esse contexto
// costuma ser cancelado assim que o comando retorna, e a goroutine abandonaria o envio.
func (m *CallManager) sendSignaling(ctx context.Context, node waBinary.Node) {
	go func() {
		sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), signalingSendTimeout)
		defer cancel()
		_, _ = m.sock.Query(sctx, node)
	}()
}

func (m *CallManager) RejectCall(ctx context.Context, callID string, reason core.EndCallReason) error {
	m.mu.Lock()
	call := m.currentCall
	if call == nil || call.CallID != callID {
		m.mu.Unlock()
		return &CallError{"no call with id " + callID}
	}
	if err := call.ApplyTransition(Transition{Type: TransitionLocalRejected, Reason: reason}); err != nil {
		m.mu.Unlock()
		return err
	}
	node := signaling.BuildRejectStanza(wanode.MustJID(call.PeerJid), call.CallID, wanode.MustJID(call.CallCreator))
	m.emitState()
	m.mu.Unlock()

	m.sendSignaling(ctx, node)
	m.cleanupMedia()
	return nil
}

func (m *CallManager) EndCall(ctx context.Context, reason core.EndCallReason) error {
	m.mu.Lock()
	call := m.currentCall
	if call == nil || call.IsEnded() {
		m.mu.Unlock()
		return nil
	}
	_ = call.ApplyTransition(Transition{Type: TransitionTerminated, Reason: reason})
	termDest := call.PeerJid
	if m.acceptedByJid != "" {
		termDest = m.acceptedByJid
	}
	node := signaling.BuildTerminateStanza(wanode.MustJID(termDest), call.CallID, wanode.MustJID(call.CallCreator))
	ended := call
	m.emitState()
	m.mu.Unlock()

	m.sendSignaling(ctx, node)
	if m.OnEnded != nil {
		m.OnEnded(ended)
	}
	m.cleanupMedia()
	return nil
}

func (m *CallManager) ownCredJid() string {
	lid := m.sock.OwnLID()
	if !lid.IsEmpty() {
		return lid.String()
	}
	return m.sock.OwnPN().String()
}

type CallError struct{ Msg string }

func (e *CallError) Error() string { return e.Msg }

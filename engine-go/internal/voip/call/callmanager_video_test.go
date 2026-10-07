package call

import (
	"context"
	"log/slog"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/media"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/transport"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

// relayEspiao registra o que o CallManager entrega ao relay, sem rede.
type relayEspiao struct {
	extraSelf, extraPeer []uint32
}

func (r *relayEspiao) SetSsrc(uint32)                          {}
func (r *relayEspiao) SetSubscriptionSsrc(uint32)              {}
func (r *relayEspiao) SetExtraSsrcs(self, peer []uint32)       { r.extraSelf, r.extraPeer = self, peer }
func (r *relayEspiao) SetOnConnected(func(string, int))        {}
func (r *relayEspiao) SetOnReceive(func([]byte))               {}
func (r *relayEspiao) ResendSubscriptions()                    {}
func (r *relayEspiao) ConfigureRelays([]transport.RelayConfig) {}
func (r *relayEspiao) Broadcast([]byte)                        {}
func (r *relayEspiao) HasConnection() bool                     { return false }
func (r *relayEspiao) ConnectedCount() int                     { return 0 }
func (r *relayEspiao) Cleanup()                                {}

// sockComNumero é o fakeSock com um número próprio, como num engine conectado.
type sockComNumero struct{ fakeSock }

func (sockComNumero) OwnPN() types.JID {
	return types.NewJID("5511988887777", types.DefaultUserServer)
}

func videoOffer(callID, from string, withVideo bool) *waBinary.Node {
	kids := []waBinary.Node{{Tag: "audio"}}
	if withVideo {
		kids = append(kids, waBinary.Node{Tag: "video"})
	}
	return &waBinary.Node{Tag: "call", Attrs: waBinary.Attrs{"from": from}, Content: []waBinary.Node{
		{Tag: "offer", Attrs: waBinary.Attrs{"call-id": callID, "call-creator": from}, Content: kids},
	}}
}

func newVideoRig(t *testing.T, withVideo bool) (*CallManager, *relayEspiao, types.JID) {
	t.Helper()
	peer := types.NewJID("5511999990001", types.DefaultUserServer)
	m := NewCallManager(&sockComNumero{}, slog.Default())
	spy := &relayEspiao{}
	m.relay = spy
	m.DeferPreaccept = true
	m.HandleCallOffer(context.Background(), videoOffer("CALLV", peer.String(), withVideo), peer)
	return m, spy, peer
}

// O relay só encaminha o vídeo do contato se o SSRC de vídeo dele estiver na alocação. O SSRC é o
// mesmo HKDF do áudio com o slot 2 (VideoSlotWord) no lugar do 0.
func TestIncomingVideoOffer_DerivesVideoSsrcsIntoRelay(t *testing.T) {
	m, spy, peer := newVideoRig(t, true)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentCall.MediaType != core.CallMediaTypeVideo {
		t.Fatalf("a oferta tem <video>, a chamada devia ser de vídeo: %v", m.currentCall.MediaType)
	}
	wantPeer := media.GenerateSecureSsrc("CALLV", peer.String(), media.VideoSlotWord)
	if m.videoPeerSsrc != wantPeer || m.videoPeerSsrc == 0 {
		t.Fatalf("SSRC de vídeo do contato = %#x, esperado %#x", m.videoPeerSsrc, wantPeer)
	}
	if m.videoPeerSsrc == m.peerSsrcs[0] {
		t.Fatal("o SSRC de vídeo não pode coincidir com o de áudio")
	}
	if len(spy.extraPeer) != 1 || spy.extraPeer[0] != wantPeer {
		t.Fatalf("o relay recebeu peer=%v, esperado [%#x]", spy.extraPeer, wantPeer)
	}
	if len(spy.extraSelf) != 1 || spy.extraSelf[0] != m.videoSelfSsrc || m.videoSelfSsrc == 0 {
		t.Fatalf("o relay recebeu self=%v, videoSelfSsrc=%#x", spy.extraSelf, m.videoSelfSsrc)
	}
}

// Chamada de voz: nenhum SSRC de vídeo, relay exatamente como antes.
func TestIncomingVoiceOffer_HasNoVideoSsrcs(t *testing.T) {
	m, spy, _ := newVideoRig(t, false)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.videoSelfSsrc != 0 || m.videoPeerSsrc != 0 {
		t.Fatalf("voz não deriva SSRC de vídeo: %#x %#x", m.videoSelfSsrc, m.videoPeerSsrc)
	}
	if len(spy.extraSelf) != 0 || len(spy.extraPeer) != 0 {
		t.Fatalf("o relay não pode receber extras na voz: %v %v", spy.extraSelf, spy.extraPeer)
	}
}

// Os SSRCs de uma chamada não podem sobrar para a seguinte.
func TestCleanupMediaClearsVideoSsrcs(t *testing.T) {
	m, _, _ := newVideoRig(t, true)
	m.cleanupMedia()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.videoSelfSsrc != 0 || m.videoPeerSsrc != 0 {
		t.Fatalf("SSRC de vídeo sobrou: %#x %#x", m.videoSelfSsrc, m.videoPeerSsrc)
	}
}

// Quando o JID do aparelho do contato é refinado (ack do relay), o SSRC de vídeo acompanha o de áudio.
func TestVideoSsrcFollowsRefinedPeerDevice(t *testing.T) {
	m, spy, _ := newVideoRig(t, true)
	m.mu.Lock()
	m.deriveVideoSsrcsLocked("", "5511999990001:7@s.whatsapp.net")
	m.applyVideoSsrcsLocked()
	want := media.GenerateSecureSsrc("CALLV", "5511999990001:7@s.whatsapp.net", media.VideoSlotWord)
	got := m.videoPeerSsrc
	m.mu.Unlock()
	if got != want || len(spy.extraPeer) != 1 || spy.extraPeer[0] != want {
		t.Fatalf("SSRC refinado = %#x, esperado %#x (relay=%v)", got, want, spy.extraPeer)
	}
}

// ---- recepção de vídeo (fase 1) ----

type videoRig struct {
	m    *CallManager
	tx   *media.SrtpSession
	aus  [][]byte
	keys []bool
	plis int
}

// newRecvRig monta um CallManager de vídeo com SRTP real e um "contato" que cifra com a chave certa.
func newRecvRig(t *testing.T) *videoRig {
	t.Helper()
	m, _, _ := newVideoRig(t, true)
	key := bytesRepeat(0x33, 32)
	km, err := media.DerivePerJidSrtpKey(key, "contato:0@lid")
	if err != nil {
		t.Fatal(err)
	}
	rx, err := media.NewSrtpSession(km, km, core.SRTPSendAuthTagLen, core.SRTPRecvAuthTagLen)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := media.NewSrtpSession(km, km, core.SRTPSendAuthTagLen, core.SRTPRecvAuthTagLen)
	if err != nil {
		t.Fatal(err)
	}
	r := &videoRig{m: m, tx: tx}
	m.mu.Lock()
	m.srtpSession = rx
	m.codec = &stubCodec{}
	m.mu.Unlock()
	m.OnPeerVideo = func(au []byte, key bool) {
		r.aus = append(r.aus, append([]byte(nil), au...))
		r.keys = append(r.keys, key)
	}
	m.OnVideoKeyframeNeeded = func() { r.plis++ }
	return r
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

type stubCodec struct{}

func (stubCodec) Encode([]float32) ([]byte, error) { return nil, nil }
func (stubCodec) Decode([]byte) ([]float32, error) { return make([]float32, 960), nil }
func (stubCodec) FrameSize() int                   { return 960 }
func (stubCodec) SampleRate() int                  { return 16000 }
func (stubCodec) Close()                           {}

// send empacota uma AU Annex-B como o WhatsApp faz (uma AU inteira, FU-A) e injeta no relay.
func (r *videoRig) send(t *testing.T, annexb []byte, idr bool, stream *media.VideoRtpStream) {
	t.Helper()
	info := media.VideoFrameInfoDelta
	if idr {
		info = media.VideoFrameInfoIDR
	}
	parts := media.PackageH264NALU(packAU(annexb))
	for i, p := range parts {
		h, _ := stream.NextPacket(i == len(parts)-1, info)
		wire, err := r.tx.Protect(&media.RtpPacket{Header: h, Payload: p})
		if err != nil {
			t.Fatal(err)
		}
		r.m.onRelayData(wire)
	}
}

// packAU monta o payload como o WhatsApp: a AU inteira vira UM NAL; os NALs são separados por
// 00 00 00 01, SEM start code no início e sem o AUD (tipo 9). Copiado do sender da meowcaller.
func packAU(annexb []byte) []byte {
	var packed []byte
	for _, n := range media.SplitAnnexB(annexb) {
		if len(n) == 0 || n[0]&0x1f == 9 {
			continue
		}
		if len(packed) > 0 {
			packed = append(packed, 0, 0, 0, 1)
		}
		packed = append(packed, n...)
	}
	return packed
}

func au(nal byte, size int) []byte {
	out := []byte{0, 0, 0, 1, nal}
	return append(out, bytesRepeat(0xAB, size)...)
}

func (r *videoRig) videoSsrc() uint32 {
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	return r.m.videoPeerSsrc
}

// O contato manda um IDR grande (vários FU-A) e depois um delta: o business recebe duas AUs completas,
// em Annex-B, com o tipo de quadro certo.
func TestRecvVideo_DeliversCompleteAccessUnits(t *testing.T) {
	r := newRecvRig(t)
	s := media.NewVideoRtpStream(r.videoSsrc(), 6000)
	idr := append(au(0x67, 20), au(0x65, 3000)...) // SPS + IDR grande
	r.send(t, idr, true, s)
	r.send(t, au(0x41, 500), false, s)
	if len(r.aus) != 2 {
		t.Fatalf("entregou %d AUs, esperado 2", len(r.aus))
	}
	if !r.keys[0] || r.keys[1] {
		t.Fatalf("flags de keyframe = %v, esperado [true false]", r.keys)
	}
	if !media.AUHasIDR(r.aus[0]) {
		t.Fatal("a 1ª AU devia conter o IDR")
	}
}

// O vídeo chega no SSRC de vídeo; o SSRC de áudio não pode ser "travado" pelo vídeo nem vice-versa.
func TestRecvVideo_DoesNotBreakAudioSubscription(t *testing.T) {
	r := newRecvRig(t)
	r.m.mu.Lock()
	before := append([]uint32(nil), r.m.peerSsrcs...)
	actual := r.m.actualPeerSet
	r.m.mu.Unlock()
	s := media.NewVideoRtpStream(r.videoSsrc(), 6000)
	r.send(t, au(0x65, 100), true, s)
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	if len(r.m.peerSsrcs) != len(before) || r.m.peerSsrcs[0] != before[0] || r.m.actualPeerSet != actual {
		t.Fatalf("o vídeo mexeu na subscrição de áudio: %v -> %v (actual %v -> %v)", before, r.m.peerSsrcs, actual, r.m.actualPeerSet)
	}
}

// Lacuna na sequência: descarta o quadro quebrado, pede keyframe uma vez e retoma no próximo IDR.
func TestRecvVideo_GapRequestsKeyframeAndRecovers(t *testing.T) {
	r := newRecvRig(t)
	s := media.NewVideoRtpStream(r.videoSsrc(), 6000)
	r.send(t, au(0x65, 100), true, s) // IDR ok
	// um quadro grande de que perdemos o pacote do meio
	parts := media.PackageH264NALU(packAU(au(0x41, 5000)))
	for i, p := range parts {
		h, _ := s.NextPacket(i == len(parts)-1, media.VideoFrameInfoDelta)
		if i == 1 {
			continue // perdido
		}
		wire, _ := r.tx.Protect(&media.RtpPacket{Header: h, Payload: p})
		r.m.onRelayData(wire)
	}
	if len(r.aus) != 1 {
		t.Fatalf("o quadro quebrado não pode ser entregue: %d AUs", len(r.aus))
	}
	if r.plis != 1 {
		t.Fatalf("pediu keyframe %d vezes, esperado 1", r.plis)
	}
	r.send(t, au(0x41, 200), false, s) // delta sem IDR: continua descartado
	if len(r.aus) != 1 {
		t.Fatal("delta antes do novo IDR não pode ser entregue")
	}
	r.send(t, au(0x65, 300), true, s) // IDR: retoma
	if len(r.aus) != 2 || !r.keys[1] {
		t.Fatalf("não retomou no IDR: %d AUs %v", len(r.aus), r.keys)
	}
}

// Pacote de vídeo repetido (cópia do mesmo pacote por outro relay) é descartado pelo anti-replay.
func TestRecvVideo_DuplicateFromAnotherRelayIsDropped(t *testing.T) {
	r := newRecvRig(t)
	s := media.NewVideoRtpStream(r.videoSsrc(), 6000)
	h, _ := s.NextPacket(true, media.VideoFrameInfoIDR)
	wire, _ := r.tx.Protect(&media.RtpPacket{Header: h, Payload: au(0x65, 50)[4:]})
	r.m.onRelayData(wire)
	r.m.onRelayData(wire)
	r.m.onRelayData(wire)
	if len(r.aus) != 1 {
		t.Fatalf("3 cópias do mesmo pacote deram %d AUs, esperado 1", len(r.aus))
	}
}

// Chamada de VOZ: o vídeo não é processado (nenhum hook disparado).
func TestRecvVideo_IgnoredInVoiceCall(t *testing.T) {
	m, _, _ := newVideoRig(t, false)
	called := false
	m.OnPeerVideo = func([]byte, bool) { called = true }
	h := media.NewRtpHeader(media.PayloadTypeH264, 1, 0, 0xB2)
	h.Marker = true
	km, _ := media.DerivePerJidSrtpKey(bytesRepeat(1, 32), "x:0@lid")
	tx, _ := media.NewSrtpSession(km, km, core.SRTPSendAuthTagLen, core.SRTPRecvAuthTagLen)
	wire, _ := tx.Protect(&media.RtpPacket{Header: h, Payload: []byte{0x65, 1, 2, 3}})
	m.onRelayData(wire)
	if called {
		t.Fatal("chamada de voz não processa vídeo")
	}
}

// Uma rajada de perdas não pode virar uma rajada de pedidos de quadro-chave: no máximo um a cada
// videoPLIInterval. Passado o intervalo, um novo pedido volta a ser permitido.
func TestRecvVideo_KeyframeRequestsAreThrottled(t *testing.T) {
	r := newRecvRig(t)
	s := media.NewVideoRtpStream(r.videoSsrc(), 6000)
	loseOne := func() {
		parts := media.PackageH264NALU(packAU(au(0x41, 5000)))
		for i, p := range parts {
			h, _ := s.NextPacket(i == len(parts)-1, media.VideoFrameInfoDelta)
			if i == 1 {
				continue
			}
			wire, _ := r.tx.Protect(&media.RtpPacket{Header: h, Payload: p})
			r.m.onRelayData(wire)
		}
	}
	r.send(t, au(0x65, 100), true, s)
	loseOne()
	r.send(t, au(0x65, 100), true, s) // recupera
	loseOne()                         // 2ª perda, logo em seguida
	if r.plis != 1 {
		t.Fatalf("2 perdas dentro do intervalo geraram %d pedidos, esperado 1", r.plis)
	}
	r.m.mu.Lock()
	r.m.videoLastPLI = r.m.videoLastPLI.Add(-2 * videoPLIInterval)
	r.m.mu.Unlock()
	r.send(t, au(0x65, 100), true, s)
	loseOne()
	if r.plis != 2 {
		t.Fatalf("passado o intervalo devia pedir de novo: %d pedidos", r.plis)
	}
}

// Vídeo que chega numa chamada de VOZ é ignorado de verdade (não só "sem hook"): nem desprotege nem
// monta quadro, e o assembler fica intacto.
func TestRecvVideo_VoiceCallDoesNotTouchTheAssembler(t *testing.T) {
	m, _, _ := newVideoRig(t, false)
	km, _ := media.DerivePerJidSrtpKey(bytesRepeat(1, 32), "x:0@lid")
	rx, _ := media.NewSrtpSession(km, km, core.SRTPSendAuthTagLen, core.SRTPRecvAuthTagLen)
	tx, _ := media.NewSrtpSession(km, km, core.SRTPSendAuthTagLen, core.SRTPRecvAuthTagLen)
	m.mu.Lock()
	m.srtpSession = rx
	m.mu.Unlock()
	delivered := 0
	m.OnPeerVideo = func([]byte, bool) { delivered++ }
	for seq := uint16(1); seq <= 3; seq++ {
		h := media.NewRtpHeader(media.PayloadTypeH264, seq, 0, 0xB2)
		h.Marker = true
		wire, _ := tx.Protect(&media.RtpPacket{Header: h, Payload: []byte{0x65, 1, 2, 3}})
		m.onRelayData(wire)
	}
	if delivered != 0 {
		t.Fatalf("voz entregou %d quadros de vídeo", delivered)
	}
	// se tivesse desprotegido, o contexto SRTP do SSRC 0xB2 teria registrado a sequência: um pacote
	// repetido seria rejeitado como replay. Aqui o contexto não pode ter sido tocado.
	h := media.NewRtpHeader(media.PayloadTypeH264, 1, 0, 0xB2)
	h.Marker = true
	wire, _ := tx.Protect(&media.RtpPacket{Header: h, Payload: []byte{0x65, 1, 2, 3}})
	if _, err := rx.Unprotect(wire); err != nil {
		t.Fatalf("a chamada de voz consumiu o contexto SRTP do vídeo: %v", err)
	}
}

// ---- orientação (CVO) ----

// sendWithInfo envia uma AU como o WhatsApp, mas com o MediaFrameInfo exato (tipo de quadro + 2 bits de CVO).
func (r *videoRig) sendWithInfo(t *testing.T, annexb []byte, info uint8, stream *media.VideoRtpStream) {
	t.Helper()
	parts := media.PackageH264NALU(packAU(annexb))
	for i, p := range parts {
		h, _ := stream.NextPacket(i == len(parts)-1, info)
		wire, err := r.tx.Protect(&media.RtpPacket{Header: h, Payload: p})
		if err != nil {
			t.Fatal(err)
		}
		r.m.onRelayData(wire)
	}
}

// O celular em retrato manda o vídeo "deitado" e avisa a rotação nos 2 bits baixos do MediaFrameInfo (CVO,
// quartos de volta horários). Sem repassar isso o navegador mostrava a imagem virada para a esquerda.
func TestRecvVideo_ReportsDisplayOrientationFromRtpExtension(t *testing.T) {
	r := newRecvRig(t)
	var got []int
	r.m.OnPeerVideoOrientation = func(q int) { got = append(got, q) }
	s := media.NewVideoRtpStream(r.videoSsrc(), 6000)

	r.sendWithInfo(t, au(0x65, 100), media.VideoFrameInfoIDR|0x03, s) // anunciado 3 → o receptor gira 1 (Android invertido)
	r.sendWithInfo(t, au(0x41, 100), media.VideoFrameInfoDelta|0x03, s)
	r.sendWithInfo(t, au(0x41, 100), media.VideoFrameInfoDelta|0x03, s)
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("orientação reportada = %v, esperado [1] (só na mudança, não a cada quadro)", got)
	}
	r.sendWithInfo(t, au(0x41, 100), media.VideoFrameInfoDelta|0x01, s) // girou o aparelho
	if len(got) != 2 || got[1] != 3 {
		t.Fatalf("mudança de orientação não reportada: %v", got)
	}
}

// A orientação acompanha o quadro: o hook de vídeo recebe a rotação junto com a AU.
func TestRecvVideo_DeliversOrientationWithTheFrame(t *testing.T) {
	r := newRecvRig(t)
	var orient []int
	r.m.OnPeerVideo = nil
	r.m.OnPeerVideoFrame = func(au []byte, key bool, q int) { orient = append(orient, q) }
	s := media.NewVideoRtpStream(r.videoSsrc(), 6000)
	r.sendWithInfo(t, au(0x65, 100), media.VideoFrameInfoIDR|0x02, s)
	r.sendWithInfo(t, au(0x41, 100), media.VideoFrameInfoDelta|0x02, s)
	if len(orient) != 2 || orient[0] != 2 || orient[1] != 2 {
		t.Fatalf("rotação por quadro = %v, esperado [2 2]", orient)
	}
}

// Sem a extensão (ou com perfil estranho) a imagem fica sem rotação, sem quebrar.
func TestRecvVideo_NoExtensionMeansNoRotation(t *testing.T) {
	r := newRecvRig(t)
	var orient []int
	r.m.OnPeerVideo = nil
	r.m.OnPeerVideoFrame = func(au []byte, key bool, q int) { orient = append(orient, q) }
	h := media.NewRtpHeader(media.PayloadTypeH264, 1, 0, r.videoSsrc())
	h.Marker = true
	wire, _ := r.tx.Protect(&media.RtpPacket{Header: h, Payload: []byte{0x65, 1, 2, 3}})
	r.m.onRelayData(wire)
	if len(orient) != 1 || orient[0] != 0 {
		t.Fatalf("sem extensão a rotação é 0: %v", orient)
	}
}

// Os quartos ímpares chegam trocados do Android; 0 e 2 não dependem do sentido do giro.
func TestReceiverRotation_SwapsOddQuartersOnly(t *testing.T) {
	for in, want := range map[int]int{0: 0, 1: 3, 2: 2, 3: 1} {
		if got := receiverRotation(in); got != want {
			t.Fatalf("receiverRotation(%d) = %d, esperado %d", in, got, want)
		}
	}
}

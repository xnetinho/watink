package call

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/alltomatos/watinkdev/engine-go/internal/voip/media"
)

// relayGravador guarda o que o CallManager manda ao relay, como se estivesse conectado.
type relayGravador struct {
	relayEspiao
	mu   sync.Mutex
	sent [][]byte
}

func (r *relayGravador) HasConnection() bool { return true }
func (r *relayGravador) Broadcast(b []byte) {
	r.mu.Lock()
	r.sent = append(r.sent, append([]byte(nil), b...))
	r.mu.Unlock()
}
func (r *relayGravador) drain() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.sent
	r.sent = nil
	return out
}

// txRig monta uma chamada de vídeo com SRTP/SRTCP reais e um "contato" que decifra com a mesma chave.
type txRig struct {
	m      *CallManager
	relay  *relayGravador
	peerRx *media.SrtpSession
	peerSk *media.SrtcpKeys
	asm    media.H264AccessUnitAssembler
	ssrc   uint32
}

func newTxRig(t *testing.T) *txRig {
	t.Helper()
	m, _, _ := newVideoRig(t, true)
	rel := &relayGravador{}
	m.relay = rel
	km, err := media.DerivePerJidSrtpKey(bytesRepeat(0x44, 32), "contato:0@lid")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := media.NewSrtpSession(km, km, core.SRTPSendAuthTagLen, core.SRTPRecvAuthTagLen)
	if err != nil {
		t.Fatal(err)
	}
	rx, err := media.NewSrtpSession(km, km, core.SRTPSendAuthTagLen, core.SRTPRecvAuthTagLen)
	if err != nil {
		t.Fatal(err)
	}
	sk, err := media.DeriveSrtcpKeys(km)
	if err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.srtpSession = tx
	m.deriveSrtcpKeysLocked(km, km)
	m.videoSelfSsrc = 0xAABB0002
	m.videoTxOn = true
	m.videoNeedIDR = true
	m.mu.Unlock()
	t.Cleanup(func() { m.mu.Lock(); m.resetVideoTxLocked(); m.mu.Unlock() })
	return &txRig{m: m, relay: rel, peerRx: rx, peerSk: sk, ssrc: 0xAABB0002}
}

type rxPacket struct {
	info   uint8
	marker bool
	ts     uint32
	frame  *uint16
}

// recv decifra os pacotes de vídeo enviados e devolve as access units remontadas e os metadados de cada
// pacote.
func (r *txRig) recv(t *testing.T, wire [][]byte) (aus [][]byte, pkts []rxPacket) {
	t.Helper()
	for _, w := range wire {
		if w[1] >= 192 && w[1] <= 223 {
			continue
		}
		p, err := r.peerRx.Unprotect(w)
		if err != nil {
			t.Fatalf("o contato não consegue decifrar o vídeo: %v", err)
		}
		if p.Header.Ssrc != r.ssrc || p.Header.PayloadType != media.PayloadTypeH264 {
			t.Fatalf("ssrc/pt errados: %x %d", p.Header.Ssrc, p.Header.PayloadType)
		}
		ext, ok := media.ParseVideoRtpExtension(p.Header)
		if !ok {
			t.Fatal("falta a extensão 0xDEBE: o WhatsApp nunca trataria o quadro como keyframe")
		}
		pkts = append(pkts, rxPacket{ext.MediaFrameInfo, p.Header.Marker, p.Header.Timestamp, ext.FrameNumber})
		if au, ok, _ := r.asm.Push(p.Header.SequenceNumber, p.Header.Marker, p.Payload); ok {
			aus = append(aus, au)
		}
	}
	return aus, pkts
}

func idrAU(size int) []byte {
	out := []byte{0, 0, 0, 1, 0x67, 1, 2, 3}
	out = append(out, 0, 0, 0, 1, 0x68, 9)
	out = append(out, 0, 0, 0, 1, 0x65)
	return append(out, bytesRepeat(0xCD, size)...)
}

func TestSendVideoFrame_DropsDeltaUntilFirstIDR(t *testing.T) {
	r := newTxRig(t)
	r.m.SendVideoFrame(au(0x41, 3000), 66*time.Millisecond)
	if n := len(r.relay.drain()); n != 0 {
		t.Fatalf("um delta antes do 1º IDR foi enviado (%d pacotes): o contato não saberia decodificar", n)
	}
	r.m.SendVideoFrame(idrAU(3000), 66*time.Millisecond)
	aus, pkts := r.recv(t, r.relay.drain())
	if len(aus) != 1 || !media.AUHasIDR(aus[0]) {
		t.Fatalf("o IDR não chegou inteiro ao contato: %d AUs", len(aus))
	}
	for _, p := range pkts {
		if p.info != media.VideoFrameInfoIDR {
			t.Fatalf("MediaFrameInfo = %#x, quer IDR %#x em todos os pacotes", p.info, media.VideoFrameInfoIDR)
		}
	}
	r.m.SendVideoFrame(au(0x41, 3000), 66*time.Millisecond)
	_, pkts = r.recv(t, r.relay.drain())
	if len(pkts) == 0 || pkts[0].info != media.VideoFrameInfoDelta {
		t.Fatal("depois do IDR o delta tem de passar, marcado como delta")
	}
}

func TestSendVideoFrame_PacketStructure(t *testing.T) {
	r := newTxRig(t)
	r.m.SendVideoFrame(idrAU(5000), 66*time.Millisecond)
	_, pkts := r.recv(t, r.relay.drain())
	if len(pkts) < 3 {
		t.Fatalf("5000 B devem virar vários FU-A, vieram %d", len(pkts))
	}
	for i, p := range pkts {
		if p.marker != (i == len(pkts)-1) {
			t.Fatalf("marker só no último pacote da AU (pacote %d)", i)
		}
		if p.ts != pkts[0].ts {
			t.Fatal("todos os pacotes de uma AU levam o mesmo timestamp")
		}
		if (p.frame != nil) != (i == 0) {
			t.Fatalf("FrameNumber só no 1º pacote da AU (pacote %d)", i)
		}
	}
	r.m.SendVideoFrame(au(0x41, 100), 100*time.Millisecond)
	_, next := r.recv(t, r.relay.drain())
	if got := next[0].ts - pkts[0].ts; got != 5940 {
		t.Fatalf("o timestamp avança pela duração do quadro anterior (66 ms = 5940 ticks): got %d", got)
	}
}

func TestSendVideoFrame_RoundTripKeepsContent(t *testing.T) {
	r := newTxRig(t)
	r.m.SendVideoFrame(idrAU(2500), 0)
	aus, _ := r.recv(t, r.relay.drain())
	if len(aus) != 1 {
		t.Fatal("sem AU")
	}
	for _, nal := range [][]byte{{0x67, 1, 2, 3}, {0x68, 9}} {
		if !bytes.Contains(aus[0], nal) {
			t.Fatalf("o NAL %x sumiu no caminho", nal)
		}
	}
	if !bytes.Contains(aus[0], bytesRepeat(0xCD, 2500)) {
		t.Fatal("o corpo do IDR foi alterado no caminho")
	}
}

func TestSendVideoFrame_CameraOffOrNoVideoSendsNothing(t *testing.T) {
	r := newTxRig(t)
	r.m.mu.Lock()
	r.m.videoTxOn = false
	r.m.mu.Unlock()
	r.m.SendVideoFrame(idrAU(500), 0)
	if len(r.relay.drain()) != 0 {
		t.Fatal("com a câmera desligada nada pode sair")
	}

	voice := newTxRig(t)
	voice.m.mu.Lock()
	voice.m.currentCall.MediaType = core.CallMediaTypeAudio
	voice.m.mu.Unlock()
	voice.m.SendVideoFrame(idrAU(500), 0)
	if len(voice.relay.drain()) != 0 {
		t.Fatal("numa chamada de voz o vídeo não pode sair")
	}
}

// O contato pede um IDR (PLI cifrado por SRTCP): o engine descarta deltas até o próximo IDR e avisa o
// codificador da câmera.
func TestPeerPLI_ForcesNextIDRAndNotifiesEncoder(t *testing.T) {
	r := newTxRig(t)
	asked := 0
	r.m.OnVideoKeyframeRequested = func() { asked++ }
	r.m.SendVideoFrame(idrAU(800), 0)
	r.relay.drain()
	r.m.SendVideoFrame(au(0x41, 800), 0)
	if len(r.relay.drain()) == 0 {
		t.Fatal("delta normal tem de passar")
	}

	pli := media.BuildPictureLossIndication(0x77770001, r.ssrc, true)
	wire, err := r.peerSk.Protect(0x77770001, 1, pli[:])
	if err != nil {
		t.Fatal(err)
	}
	r.m.onRelayData(wire)
	if asked != 1 {
		t.Fatalf("o codificador foi avisado %d vez(es), quer 1", asked)
	}
	r.m.SendVideoFrame(au(0x41, 800), 0)
	if n := len(r.relay.drain()); n != 0 {
		t.Fatalf("depois do PLI um delta saiu (%d pacotes)", n)
	}
	r.m.SendVideoFrame(idrAU(800), 0)
	if len(r.relay.drain()) == 0 {
		t.Fatal("o IDR depois do PLI tem de passar")
	}
	r.m.SendVideoFrame(au(0x41, 800), 0)
	if len(r.relay.drain()) == 0 {
		t.Fatal("e o fluxo volta ao normal depois dele")
	}
}

func TestPeerRTCP_ForgedOrForOtherSSRCIsIgnored(t *testing.T) {
	r := newTxRig(t)
	asked := 0
	r.m.OnVideoKeyframeRequested = func() { asked++ }
	r.m.SendVideoFrame(idrAU(300), 0)
	r.relay.drain()

	pli := media.BuildPictureLossIndication(1, r.ssrc, true)
	wire, _ := r.peerSk.Protect(1, 1, pli[:])
	wire[len(wire)-1] ^= 1
	r.m.onRelayData(wire)

	other := media.BuildPictureLossIndication(1, 0x12121212, true)
	wire2, _ := r.peerSk.Protect(1, 2, other[:])
	r.m.onRelayData(wire2)

	if asked != 0 {
		t.Fatal("PLI adulterado ou para outro SSRC não pode forçar IDR")
	}
	r.m.SendVideoFrame(au(0x41, 300), 0)
	if len(r.relay.drain()) == 0 {
		t.Fatal("o delta continua passando")
	}
}

// O SR+SDES periódico é o que faz o vídeo de quem envia começar a fluir para o outro lado.
func TestSenderReportWithSdes_IsAuthenticatedAndCarriesCounters(t *testing.T) {
	r := newTxRig(t)
	r.m.SendVideoFrame(idrAU(2000), 0)
	r.relay.drain()

	r.m.sendVideoSenderReport(time.UnixMilli(1718000000000))
	wire := r.relay.drain()
	if len(wire) != 1 || len(wire[0]) != 74 {
		t.Fatalf("o SR+SDES protegido tem 74 bytes (60 + 14), veio %d pacotes", len(wire))
	}
	plain, index, err := r.peerSk.Unprotect(r.ssrc, wire[0])
	if err != nil {
		t.Fatalf("o contato não autentica o SR: %v", err)
	}
	if index != 1 || plain[0] != 0x90 || plain[1] != 200 || plain[28] != 0x91 || plain[29] != 202 {
		t.Fatalf("SR/SDES malformado: idx=%d %x %x", index, plain[:2], plain[28:30])
	}
	r.m.mu.Lock()
	pk, oc := r.m.videoPackets, r.m.videoOctets
	r.m.mu.Unlock()
	if pk == 0 || oc == 0 {
		t.Fatal("o contador de pacotes/octetos do SR ficou zerado")
	}
	r.m.sendVideoSenderReport(time.UnixMilli(1718000001500))
	if _, i2, err := r.peerSk.Unprotect(r.ssrc, r.relay.drain()[0]); err != nil || i2 != 2 {
		t.Fatalf("o índice SRTCP precisa crescer a cada pacote: %d %v", i2, err)
	}
}

func TestSenderReport_NothingBeforeFirstFrame(t *testing.T) {
	r := newTxRig(t)
	r.m.sendVideoSenderReport(time.Now())
	if len(r.relay.drain()) != 0 {
		t.Fatal("sem quadro enviado não há o que reportar")
	}
}

func TestResetVideoTx_StopsReportsAndClearsState(t *testing.T) {
	r := newTxRig(t)
	r.m.SendVideoFrame(idrAU(300), 0)
	r.m.mu.Lock()
	running := r.m.rtcpStop != nil
	r.m.mu.Unlock()
	if !running {
		t.Fatal("o laço do SR deveria estar rodando depois do 1º quadro")
	}
	r.m.cleanupMedia()
	r.m.mu.Lock()
	defer r.m.mu.Unlock()
	if r.m.rtcpStop != nil || r.m.videoTx != nil || r.m.videoTxOn || r.m.srtcpSend != nil || r.m.videoPackets != 0 {
		t.Fatal("cleanupMedia não zerou o estado de envio de vídeo")
	}
}

func TestPackAccessUnit_DropsAUDAndJoinsNALs(t *testing.T) {
	in := append([]byte{0, 0, 0, 1, 0x09, 0xF0}, idrAU(10)...)
	got := media.PackAccessUnit(in)
	if got[0] != 0x67 {
		t.Fatalf("o AUD (tipo 9) sai e não pode sobrar start code no início: %x", got[:6])
	}
	if !bytes.Equal(got, packAU(in)) {
		t.Fatal("PackAccessUnit diverge do montador de referência usado nos testes de recepção")
	}
	if media.PackAccessUnit([]byte{0, 0, 0, 1, 0x09, 0xF0}) != nil {
		t.Fatal("só AUD não é um quadro")
	}
}

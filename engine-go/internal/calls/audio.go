package calls

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/media"
)

const (
	// frameBytes: 20 ms de PCM 16 kHz mono Int16 LE (320 amostras).
	frameBytes = 640
	// outQueueFrames: no máximo 1 s de áudio na fila rumo ao business. Cheia, o
	// quadro mais antigo é descartado: atraso acumulado é pior que um pico de ruído.
	outQueueFrames = 50
	// audioGrace: sem o canal de áudio por esse tempo numa chamada atendida, o
	// engine a encerra (o business já encerra antes, aos 10 s; isto é a rede de segurança).
	audioGrace = 30 * time.Second
)

var (
	ErrAudioInUse = errors.New("a chamada já tem um canal de áudio aberto")
	ErrNoMedia    = errors.New("chamada sem mídia disponível")
)

// Telemetry é a medição de qualidade enviada a 1 Hz. A perda é só a do sentido
// contato→operador: sem RTCP, o sentido contrário não é mensurável.
type Telemetry struct {
	Type           string  `json:"type"`
	CallID         string  `json:"callId"`
	RttMs          *int    `json:"rttMs"`
	LossPct        float64 `json:"lossPct"`
	JitterMs       float64 `json:"jitterMs"`
	TxBytesPerSec  float64 `json:"txBytesPerSec"`
	RxBytesPerSec  float64 `json:"rxBytesPerSec"`
	TxLevel        float64 `json:"txLevel"`
	RxLevel        float64 `json:"rxLevel"`
	RelayConnected bool    `json:"relayConnected"`
	RelayDrops     int     `json:"relayDrops"`
	SilentMs       int64   `json:"silentMs"`
	NoPeerAudio    bool    `json:"noPeerAudio"`
	DroppedFrames  int64   `json:"droppedFrames"`
	At             int64   `json:"at"`
}

// AudioPipe é o canal de áudio de UMA chamada: PCM do contato para o business
// (Out), PCM do operador para o contato (WritePCM) e a telemetria (Telemetry).
type AudioPipe struct {
	s         *Session
	ac        *activeCall
	out       chan []byte
	tel       chan Telemetry
	done      chan struct{}
	once      sync.Once
	dropped   atomic.Int64
	tickEvery time.Duration
}

func (p *AudioPipe) Out() <-chan []byte          { return p.out }
func (p *AudioPipe) Telemetry() <-chan Telemetry { return p.tel }
func (p *AudioPipe) Done() <-chan struct{}       { return p.done }
func (p *AudioPipe) QueueLen() int               { return len(p.out) }

// pushPeerPCM entrega o áudio do contato. NUNCA bloqueia (roda na goroutine do
// relay): com a fila cheia descarta o quadro mais antigo.
func (p *AudioPipe) pushPeerPCM(pcm []float32) {
	b := media.PCMFloat32ToInt16LE(pcm)
	for len(b) >= frameBytes {
		p.pushFrame(b[:frameBytes:frameBytes])
		b = b[frameBytes:]
	}
}

func (p *AudioPipe) pushFrame(f []byte) {
	select {
	case <-p.done:
		return
	default:
	}
	for {
		select {
		case p.out <- f:
			return
		default:
			select {
			case <-p.out:
				p.dropped.Add(1)
			default:
			}
		}
	}
}

// WritePCM recebe áudio do operador (qualquer múltiplo de 2 bytes, em geral 640)
// e o entrega ao codec. Entrada ímpar ou vazia é ignorada.
func (p *AudioPipe) WritePCM(b []byte) {
	if len(b) < 2 {
		return
	}
	b = b[:len(b)&^1]
	pcm := media.PCMInt16LEToFloat32(b)
	p.ac.txMeter.add(pcm, 0)
	p.ac.h.FeedPCM(pcm)
}

// Close libera o canal (idempotente). Se a chamada continua atendida, arma a
// rede de segurança de audioGrace.
func (p *AudioPipe) Close() {
	p.once.Do(func() {
		close(p.done)
		p.s.releasePipe(p.ac, p)
	})
}

func (p *AudioPipe) loop() {
	t := time.NewTicker(p.tickEvery)
	defer t.Stop()
	last := time.Now()
	for {
		select {
		case <-p.done:
			return
		case now := <-t.C:
			tel := p.s.measure(p.ac, now.Sub(last), p.dropped.Load())
			last = now
			select {
			case p.tel <- tel:
			default:
				select {
				case <-p.tel:
				default:
				}
				p.tel <- tel
			}
			p.s.publishQuality(tel)
		}
	}
}

// OpenAudio abre o canal de áudio da chamada. No máximo um por chamada.
func (s *Session) OpenAudio(callID string) (*AudioPipe, error) {
	s.mu.Lock()
	ac := s.active
	if ac == nil || ac.id != callID || ac.ended {
		s.mu.Unlock()
		return nil, ErrNoCall
	}
	if ac.pipe != nil {
		s.mu.Unlock()
		return nil, ErrAudioInUse
	}
	p := &AudioPipe{s: s, ac: ac, out: make(chan []byte, outQueueFrames), tel: make(chan Telemetry, 4),
		done: make(chan struct{}), tickEvery: s.tickEvery}
	ac.pipe = p
	if ac.grace != nil {
		ac.grace.Stop()
		ac.grace = nil
	}
	s.mu.Unlock()
	go p.loop()
	return p, nil
}

func (s *Session) releasePipe(ac *activeCall, p *AudioPipe) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ac.pipe == p {
		ac.pipe = nil
	}
	if ac.ended || ac.grace != nil || !(ac.accepted || !ac.incoming) {
		return
	}
	ac.grace = s.clock.AfterFunc(audioGrace, func() { s.onAudioGrace(ac) })
}

func (s *Session) onAudioGrace(ac *activeCall) {
	s.mu.Lock()
	skip := ac.ended || ac.pipe != nil
	s.mu.Unlock()
	if !skip {
		s.run(ac, func() { _ = ac.h.End(context.Background(), ReasonFailed) })
	}
}

// wireMedia liga as medições e o áudio do contato ao canal da chamada.
func (s *Session) wireMedia(ac *activeCall) {
	ac.h.SetMedia(MediaHooks{
		OnPeerPCM: func(pcm []float32) {
			ac.rxMeter.add(pcm, 0)
			s.mu.Lock()
			p := ac.pipe
			s.mu.Unlock()
			if p != nil {
				p.pushPeerPCM(pcm)
			}
		},
		OnPeerRtp: func(seq uint16, ts uint32, payloadLen int) {
			ac.rx.Observe(seq, ts, time.Now())
			ac.rxMeter.add(nil, payloadLen)
		},
		OnSentRtp: func(size int) { ac.txMeter.add(nil, size) },
	})
}

func (s *Session) measure(ac *activeCall, window time.Duration, dropped int64) Telemetry {
	loss, jitter, _ := ac.rx.Snapshot()
	txB, txL := ac.txMeter.drain(window)
	rxB, rxL := ac.rxMeter.drain(window)
	up := ac.h.RelayConnected()
	if ac.relayWasUp && !up {
		ac.relayDrops++
	}
	ac.relayWasUp = up
	silent := ac.rx.SilentFor(time.Now())
	tel := Telemetry{
		Type: "quality", CallID: ac.id, LossPct: loss * 100, JitterMs: jitter,
		TxBytesPerSec: txB, RxBytesPerSec: rxB, TxLevel: txL, RxLevel: rxL,
		RelayConnected: up, RelayDrops: ac.relayDrops, SilentMs: silent.Milliseconds(),
		NoPeerAudio: silent > silenceAlert, DroppedFrames: dropped, At: time.Now().UnixMilli(),
	}
	if rtt, ok := ac.h.RelayRTTMs(); ok {
		tel.RttMs = &rtt
	}
	return tel
}

func (s *Session) publishQuality(t Telemetry) {
	raw, err := json.Marshal(t)
	if err != nil {
		return
	}
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	delete(m, "type")
	s.emit("call.quality", m)
}

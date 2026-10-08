package calls

import (
	"math"
	"sync"
	"time"
)

const (
	rtpClockRate = 16000
	// silenceAlert: tempo sem áudio do contato a partir do qual a telemetria avisa.
	silenceAlert = 5 * time.Second
)

// RxStats mede o áudio RECEBIDO do contato a partir do cabeçalho RTP: perda por
// lacuna de sequência e jitter (RFC 3550, 6.4.1). Não há RTCP, então a perda no
// sentido contrário (operador→contato) NÃO é mensurável — a UI precisa dizer isso.
type RxStats struct {
	mu          sync.Mutex
	started     bool
	baseSeq     uint32
	maxSeq      uint16
	cycles      uint32
	received    uint32
	jitter      float64
	lastTransit float64
	lastArrival time.Time
}

// Observe registra um pacote RTP (seq, timestamp RTP) que chegou em `at`.
func (r *RxStats) Observe(seq uint16, ts uint32, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started {
		r.started = true
		r.baseSeq = uint32(seq)
		r.maxSeq = seq
	} else {
		delta := seq - r.maxSeq
		switch {
		case delta == 0:
			return
		case delta < 0x8000:
			if seq < r.maxSeq {
				r.cycles += 1 << 16
			}
			r.maxSeq = seq
		default:
			// reordenado/atrasado: conta como recebido, não move o máximo.
		}
	}
	r.received++
	r.lastArrival = at

	arrival := float64(at.UnixNano()) / 1e9 * rtpClockRate
	transit := arrival - float64(ts)
	if r.received > 1 {
		d := math.Abs(transit - r.lastTransit)
		r.jitter += (d - r.jitter) / 16
	}
	r.lastTransit = transit
}

// Snapshot devolve perda (0..1) e jitter (ms) acumulados.
func (r *RxStats) Snapshot() (loss float64, jitterMs float64, received uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started {
		return 0, 0, 0
	}
	expected := r.cycles + uint32(r.maxSeq) - r.baseSeq + 1
	if expected > r.received {
		loss = float64(expected-r.received) / float64(expected)
	}
	return loss, r.jitter / rtpClockRate * 1000, r.received
}

// SilentFor diz há quanto tempo não chega áudio do contato (0 se nunca chegou).
func (r *RxStats) SilentFor(now time.Time) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started {
		return 0
	}
	return now.Sub(r.lastArrival)
}

// rms devolve o nível do quadro em 0..1.
func rms(pcm []float32) float64 {
	if len(pcm) == 0 {
		return 0
	}
	var sum float64
	for _, s := range pcm {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(pcm)))
}

// meter acumula bytes e nível de áudio de um sentido; o 1 Hz lê e zera a janela.
type meter struct {
	mu    sync.Mutex
	bytes int
	sumSq float64
	n     int
}

func (m *meter) add(pcm []float32, wireBytes int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bytes += wireBytes
	for _, s := range pcm {
		m.sumSq += float64(s) * float64(s)
	}
	m.n += len(pcm)
}

// drain devolve bytes/s e nível RMS da janela e a reinicia.
func (m *meter) drain(window time.Duration) (bytesPerSec float64, level float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if window > 0 {
		bytesPerSec = float64(m.bytes) / window.Seconds()
	}
	if m.n > 0 {
		level = math.Sqrt(m.sumSq / float64(m.n))
	}
	m.bytes, m.sumSq, m.n = 0, 0, 0
	return
}

package calls

import (
	"math"
	"testing"
	"time"
)

var t0 = time.Unix(1_700_000_000, 0)

func feed(r *RxStats, seqs []uint16, step time.Duration, tsStep uint32, jit func(i int) time.Duration) {
	for i, s := range seqs {
		r.Observe(s, uint32(i)*tsStep, t0.Add(time.Duration(i)*step+jit(i)))
	}
}

func seqRange(a, n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		out[i] = uint16(a + i)
	}
	return out
}

func noJit(int) time.Duration { return 0 }

func TestRx_NoLossNoJitter(t *testing.T) {
	var r RxStats
	feed(&r, seqRange(100, 200), 60*time.Millisecond, 960, noJit)
	loss, jit, n := r.Snapshot()
	if loss != 0 || n != 200 || jit > 0.01 {
		t.Fatalf("loss=%v jitter=%v n=%d", loss, jit, n)
	}
}

func TestRx_TenPercentLoss(t *testing.T) {
	var r RxStats
	var seqs []uint16
	for i := 0; i < 1000; i++ {
		if i%10 != 5 {
			seqs = append(seqs, uint16(i))
		}
	}
	for _, s := range seqs {
		r.Observe(s, uint32(s)*960, t0.Add(time.Duration(s)*60*time.Millisecond))
	}
	loss, _, _ := r.Snapshot()
	if math.Abs(loss-0.10) > 0.005 {
		t.Fatalf("perda=%v, esperava ~0.10", loss)
	}
}

func TestRx_ReorderingIsNotLoss(t *testing.T) {
	var r RxStats
	for i, s := range []uint16{1, 2, 4, 3, 5, 6, 8, 7, 9, 10} {
		r.Observe(s, uint32(i)*960, t0.Add(time.Duration(i)*60*time.Millisecond))
	}
	loss, _, n := r.Snapshot()
	if loss != 0 || n != 10 {
		t.Fatalf("loss=%v n=%d", loss, n)
	}
}

func TestRx_DuplicateIsIgnored(t *testing.T) {
	var r RxStats
	r.Observe(1, 0, t0)
	r.Observe(1, 0, t0.Add(time.Millisecond))
	r.Observe(2, 960, t0.Add(60*time.Millisecond))
	if _, _, n := r.Snapshot(); n != 2 {
		t.Fatalf("n=%d", n)
	}
}

func TestRx_SequenceWraparound(t *testing.T) {
	var r RxStats
	var seqs []uint16
	for i := 0; i < 20; i++ {
		seqs = append(seqs, uint16(65530+i))
	}
	feed(&r, seqs, 60*time.Millisecond, 960, noJit)
	loss, _, n := r.Snapshot()
	if loss != 0 || n != 20 {
		t.Fatalf("na virada de 65535→0: loss=%v n=%d", loss, n)
	}
}

func TestRx_KnownJitter(t *testing.T) {
	var r RxStats
	// chegada alterna +/-10 ms em torno do ritmo ideal: o trânsito oscila 20 ms
	// (320 amostras a 16 kHz), então o jitter converge para ~20 ms * (16/16 ...)
	jit := func(i int) time.Duration {
		if i%2 == 0 {
			return -10 * time.Millisecond
		}
		return 10 * time.Millisecond
	}
	feed(&r, seqRange(0, 600), 60*time.Millisecond, 960, jit)
	_, j, _ := r.Snapshot()
	if math.Abs(j-20) > 1.5 {
		t.Fatalf("jitter=%vms, esperava ~20ms", j)
	}
}

func TestRx_StableArrivalsGiveNearZeroJitter(t *testing.T) {
	var r RxStats
	feed(&r, seqRange(0, 300), 60*time.Millisecond, 960, noJit)
	if _, j, _ := r.Snapshot(); j > 0.05 {
		t.Fatalf("jitter=%v", j)
	}
}

func TestRx_SilentFor(t *testing.T) {
	var r RxStats
	if r.SilentFor(t0) != 0 {
		t.Fatal("sem nenhum pacote não há silêncio a medir")
	}
	r.Observe(1, 0, t0)
	if got := r.SilentFor(t0.Add(6 * time.Second)); got != 6*time.Second {
		t.Fatalf("silêncio=%v", got)
	}
}

func TestRMS(t *testing.T) {
	if rms(nil) != 0 {
		t.Fatal("vazio")
	}
	if got := rms([]float32{0, 0, 0}); got != 0 {
		t.Fatalf("silêncio=%v", got)
	}
	if got := rms([]float32{0.5, -0.5, 0.5, -0.5}); math.Abs(got-0.5) > 1e-6 {
		t.Fatalf("onda quadrada 0.5 → %v", got)
	}
	sine := make([]float32, 16000)
	for i := range sine {
		sine[i] = float32(math.Sin(2 * math.Pi * 440 * float64(i) / 16000))
	}
	if got := rms(sine); math.Abs(got-math.Sqrt2/2) > 0.01 {
		t.Fatalf("seno de amplitude 1 → %v, esperava ~0.707", got)
	}
}

func TestMeter_BitrateAndLevel(t *testing.T) {
	var m meter
	m.add([]float32{0.5, -0.5, 0.5, -0.5}, 400)
	m.add([]float32{0.5, -0.5, 0.5, -0.5}, 400)
	bps, lvl := m.drain(2 * time.Second)
	if bps != 400 || math.Abs(lvl-0.5) > 1e-6 {
		t.Fatalf("bytes/s=%v nível=%v", bps, lvl)
	}
	bps, lvl = m.drain(time.Second)
	if bps != 0 || lvl != 0 {
		t.Fatalf("a janela deve zerar: %v %v", bps, lvl)
	}
}

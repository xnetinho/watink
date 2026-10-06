package calls

import (
	"math"
	"testing"
)

func p(v float64) *float64 { return &v }

func TestAssess_Levels(t *testing.T) {
	cases := []struct {
		name   string
		in     Sample
		level  int
		alerts []string
	}{
		{"tudo bom", Sample{RttMs: p(50), LossPct: 0, JitterMs: 5}, LevelGood, nil},
		{"sem rtt medido", Sample{LossPct: 0, JitterMs: 5}, LevelGood, nil},
		{"rtt no limite de regular (200) ainda bom", Sample{RttMs: p(200)}, LevelGood, nil},
		{"rtt logo acima de 200 é regular", Sample{RttMs: p(201)}, LevelFair, nil},
		{"rtt no limite de ruim (400) ainda regular", Sample{RttMs: p(400)}, LevelFair, nil},
		{"rtt acima de 400 é ruim e alerta", Sample{RttMs: p(401)}, LevelPoor, []string{AlertRtt}},
		{"perda 2% ainda bom", Sample{LossPct: 2}, LevelGood, nil},
		{"perda acima de 2% é regular", Sample{LossPct: 2.1}, LevelFair, nil},
		{"perda 5% ainda regular", Sample{LossPct: 5}, LevelFair, nil},
		{"perda acima de 5% é ruim e alerta", Sample{LossPct: 5.1}, LevelPoor, []string{AlertLoss}},
		{"jitter 30 ainda bom", Sample{JitterMs: 30}, LevelGood, nil},
		{"jitter acima de 30 é regular", Sample{JitterMs: 30.5}, LevelFair, nil},
		{"jitter 60 ainda regular", Sample{JitterMs: 60}, LevelFair, nil},
		{"jitter acima de 60 é ruim e alerta", Sample{JitterMs: 61}, LevelPoor, []string{AlertJitter}},
		{"o pior fator manda", Sample{RttMs: p(250), LossPct: 6, JitterMs: 10}, LevelPoor, []string{AlertLoss}},
		{"vários alertas", Sample{RttMs: p(500), LossPct: 9, JitterMs: 90}, LevelPoor, []string{AlertRtt, AlertLoss, AlertJitter}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Assess(c.in)
			if got.Level != c.level {
				t.Fatalf("nível=%d, esperava %d", got.Level, c.level)
			}
			if len(got.Alerts) != len(c.alerts) {
				t.Fatalf("alertas=%v, esperava %v", got.Alerts, c.alerts)
			}
			for i := range c.alerts {
				if got.Alerts[i] != c.alerts[i] {
					t.Fatalf("alertas=%v, esperava %v", got.Alerts, c.alerts)
				}
			}
		})
	}
}

func TestEstimateMOS_KnownPoints(t *testing.T) {
	// Valores calculados à mão com R = 93,2 − Id − Ie (Id = 0,024·d, +0,11·(d−177,3)
	// acima de 177,3 ms; Ie = 30·ln(1+15·perda)) e conferidos com a tabela do G.107:
	// R=93,2→4,41; R=80→4,0; R=70→3,6; R=60→3,1; R=50→2,6.
	cases := []struct {
		name string
		in   Sample
		want float64
	}{
		{"rede perfeita (R=93,2)", Sample{RttMs: p(0)}, 4.4},
		{"rtt 100ms (R=92,0)", Sample{RttMs: p(100)}, 4.4},
		{"rtt 300ms jitter 20 (R=87,2)", Sample{RttMs: p(300), JitterMs: 20}, 4.3},
		{"perda 5% (R=75,8)", Sample{RttMs: p(50), LossPct: 5}, 3.9},
		{"perda 10% (R=65,1)", Sample{RttMs: p(50), LossPct: 10}, 3.4},
		{"perda 30% (R=41,5)", Sample{RttMs: p(50), LossPct: 30}, 2.1},
		{"rtt 800ms (R=59,1, atraso alto)", Sample{RttMs: p(800)}, 3.1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EstimateMOS(c.in); math.Abs(got-c.want) > 0.11 {
				t.Fatalf("MOS=%v, esperava ~%v", got, c.want)
			}
		})
	}
}

func TestEstimateMOS_BoundsAndMonotonic(t *testing.T) {
	for loss := 0.0; loss <= 100; loss += 5 {
		for _, rtt := range []float64{0, 100, 500, 5000} {
			m := EstimateMOS(Sample{RttMs: p(rtt), LossPct: loss, JitterMs: 40})
			if m < 1 || m > 4.5 {
				t.Fatalf("MOS fora de 1..4.5: %v (perda %v, rtt %v)", m, loss, rtt)
			}
		}
	}
	prev := 5.0
	for loss := 0.0; loss <= 50; loss += 5 {
		m := EstimateMOS(Sample{RttMs: p(80), LossPct: loss})
		if m > prev+1e-9 {
			t.Fatalf("mais perda não pode melhorar o índice: %v > %v", m, prev)
		}
		prev = m
	}
	if EstimateMOS(Sample{LossPct: -3}) != EstimateMOS(Sample{}) {
		t.Fatal("perda negativa deve ser tratada como zero")
	}
}

func TestSummary_AveragesAndWorst(t *testing.T) {
	var s Summary
	if r := s.Result(); r.Mos != nil || r.RttAvg != nil || r.LossAvg != nil {
		t.Fatal("sem medição o resumo é todo nulo")
	}
	s.Add(Sample{RttMs: p(100), LossPct: 0, JitterMs: 10})
	s.Add(Sample{RttMs: p(300), LossPct: 4, JitterMs: 50})
	s.Add(Sample{LossPct: 2, JitterMs: 30})
	r := s.Result()
	check := func(name string, got *float64, want float64) {
		t.Helper()
		if got == nil || math.Abs(*got-want) > 0.051 {
			t.Fatalf("%s=%v, esperava %v", name, got, want)
		}
	}
	check("rttAvg", r.RttAvg, 200)
	check("rttMax", r.RttMax, 300)
	check("lossAvg", r.LossAvg, 2)
	check("lossMax", r.LossMax, 4)
	check("jitterAvg", r.JitterAvg, 30)
	check("jitterMax", r.JitterMax, 50)
	if r.Mos == nil || *r.Mos < 1 || *r.Mos > 4.5 {
		t.Fatalf("mos=%v", r.Mos)
	}
}

func TestSummary_NoRttMeasuredLeavesRttNull(t *testing.T) {
	var s Summary
	s.Add(Sample{LossPct: 1, JitterMs: 5})
	r := s.Result()
	if r.RttAvg != nil || r.RttMax != nil {
		t.Fatal("RTT nunca medido deve ficar nulo, não zero")
	}
	if r.LossAvg == nil {
		t.Fatal("a perda foi medida")
	}
}

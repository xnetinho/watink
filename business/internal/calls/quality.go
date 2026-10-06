// Package calls reúne as regras de negócio das chamadas de voz do WhatsApp no
// business: qualidade, elegibilidade, registro e gravação. O engine só mede e
// transporta; aqui se decide.
package calls

import "math"

// Níveis do indicador de sinal.
const (
	LevelGood = 3
	LevelFair = 2
	LevelPoor = 1
)

// Limites de alerta (decisão de produto, ajustáveis aqui sem mexer no engine).
// Acima de qualquer um deles o nível cai para o estágio correspondente.
const (
	RttFairMs, RttPoorMs       = 200.0, 400.0
	LossFairPct, LossPoorPct   = 2.0, 5.0
	JitterFairMs, JitterPoorMs = 30.0, 60.0
)

// Sample é uma medição de 1 s vinda do engine. RttMs nulo = não medido.
type Sample struct {
	RttMs    *float64
	LossPct  float64
	JitterMs float64
}

// Assessment é a interpretação de uma medição.
type Assessment struct {
	Level int
	// Mos é uma ESTIMATIVA (modelo E simplificado ITU-T G.107) a partir de rede.
	// Não é a qualidade percebida pelo contato.
	Mos    float64
	Alerts []string
}

// Códigos de alerta devolvidos em Assessment.Alerts.
const (
	AlertRtt    = "rtt"
	AlertLoss   = "loss"
	AlertJitter = "jitter"
)

// Assess traduz uma medição em nível, índice estimado e alertas.
func Assess(s Sample) Assessment {
	level := LevelGood
	var alerts []string
	lower := func(to int) {
		if to < level {
			level = to
		}
	}
	if s.RttMs != nil {
		switch {
		case *s.RttMs > RttPoorMs:
			lower(LevelPoor)
			alerts = append(alerts, AlertRtt)
		case *s.RttMs > RttFairMs:
			lower(LevelFair)
		}
	}
	switch {
	case s.LossPct > LossPoorPct:
		lower(LevelPoor)
		alerts = append(alerts, AlertLoss)
	case s.LossPct > LossFairPct:
		lower(LevelFair)
	}
	switch {
	case s.JitterMs > JitterPoorMs:
		lower(LevelPoor)
		alerts = append(alerts, AlertJitter)
	case s.JitterMs > JitterFairMs:
		lower(LevelFair)
	}
	return Assessment{Level: level, Mos: EstimateMOS(s), Alerts: alerts}
}

// EstimateMOS aplica o modelo E simplificado: R = 93,2 − Id − Ie,eff, com Id o
// atraso (RTT/2 + jitter como folga do buffer) e Ie,eff a perda. Resultado em
// 1,0–4,5, arredondado a 0,1. É uma estimativa só de rede; o codec não entra.
func EstimateMOS(s Sample) float64 {
	oneWay := s.JitterMs * 2
	if s.RttMs != nil {
		oneWay += *s.RttMs / 2
	}
	id := 0.024 * oneWay
	if oneWay > 177.3 {
		id += 0.11 * (oneWay - 177.3)
	}
	loss := math.Max(s.LossPct, 0)
	ie := 30 * math.Log(1+15*loss/100)
	r := 93.2 - id - ie
	return round1(rToMOS(r))
}

func rToMOS(r float64) float64 {
	switch {
	case r <= 0:
		return 1
	case r >= 100:
		return 4.5
	}
	return 1 + 0.035*r + r*(r-60)*(100-r)*7e-6
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// Summary acumula o resumo agregado de uma chamada (sem série temporal).
type Summary struct {
	N                    int
	RttSum, RttMax       float64
	RttN                 int
	LossSum, LossMax     float64
	JitterSum, JitterMax float64
	MosSum               float64
}

// Add inclui uma medição no resumo.
func (m *Summary) Add(s Sample) {
	m.N++
	if s.RttMs != nil {
		m.RttN++
		m.RttSum += *s.RttMs
		m.RttMax = math.Max(m.RttMax, *s.RttMs)
	}
	m.LossSum += s.LossPct
	m.LossMax = math.Max(m.LossMax, s.LossPct)
	m.JitterSum += s.JitterMs
	m.JitterMax = math.Max(m.JitterMax, s.JitterMs)
	m.MosSum += EstimateMOS(s)
}

// Result são as médias e os piores valores, nulos quando não houve medição.
type Result struct {
	RttAvg, RttMax, LossAvg, LossMax, JitterAvg, JitterMax, Mos *float64
}

// Result calcula médias e máximos; vazio se nada foi medido. O RTT só entra se
// foi medido em algum momento.
func (m *Summary) Result() Result {
	if m.N == 0 {
		return Result{}
	}
	f := func(v float64) *float64 { v = round1(v); return &v }
	n := float64(m.N)
	r := Result{
		LossAvg: f(m.LossSum / n), LossMax: f(m.LossMax),
		JitterAvg: f(m.JitterSum / n), JitterMax: f(m.JitterMax),
		Mos: f(m.MosSum / n),
	}
	if m.RttN > 0 {
		r.RttAvg, r.RttMax = f(m.RttSum/float64(m.RttN)), f(m.RttMax)
	}
	return r
}

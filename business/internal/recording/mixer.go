// Package recording grava chamadas de voz em MP3 dentro do business: soma o áudio
// do operador e do contato num único canal mono e o codifica com o shine-mp3.
package recording

import (
	"encoding/binary"
)

const (
	// SampleRate do áudio das chamadas.
	SampleRate = 16000
	// FrameSamples é o tamanho do quadro do navegador/engine: 20 ms.
	FrameSamples = 320
	maxPeak      = 32767
)

// toSamples converte PCM Int16 little-endian em amostras.
func toSamples(b []byte) []int16 {
	out := make([]int16, len(b)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(b[i*2:]))
	}
	return out
}

// Mixer soma dois sentidos (operador e contato) em um canal mono por RELÓGIO: cada
// quadro de 20 ms ocupa o seu instante, e o silêncio de um lado vira zero em vez de
// deslocar o outro. Quando um lado atrasa, o outro não espera.
//
// Não é seguro para uso concorrente: quem chama serializa (o gravador usa um mutex).
type Mixer struct {
	op, peer []int16
	out      []int16
}

// AddOperator e AddPeer enfileiram um quadro PCM (Int16 LE) de cada sentido.
func (m *Mixer) AddOperator(pcm []byte) { m.op = append(m.op, toSamples(pcm)...) }
func (m *Mixer) AddPeer(pcm []byte)     { m.peer = append(m.peer, toSamples(pcm)...) }

// Tick consome um quadro (FrameSamples) de cada lado, soma com limitação de pico e
// põe o resultado na saída. Lado sem áudio vira silêncio.
func (m *Mixer) Tick() {
	for i := 0; i < FrameSamples; i++ {
		var a, b int32
		if i < len(m.op) {
			a = int32(m.op[i])
		}
		if i < len(m.peer) {
			b = int32(m.peer[i])
		}
		m.out = append(m.out, clip(a+b))
	}
	m.op = drop(m.op, FrameSamples)
	m.peer = drop(m.peer, FrameSamples)
}

func drop(s []int16, n int) []int16 {
	if len(s) <= n {
		return s[:0]
	}
	return s[n:]
}

// clip limita a soma ao intervalo do Int16 sem dar a volta (distorção grosseira).
func clip(v int32) int16 {
	switch {
	case v > maxPeak:
		return maxPeak
	case v < -maxPeak-1:
		return -maxPeak - 1
	}
	return int16(v)
}

// Take devolve (e remove) até n amostras prontas; vazio se ainda não há n.
func (m *Mixer) Take(n int) []int16 {
	if len(m.out) < n {
		return nil
	}
	out := make([]int16, n)
	copy(out, m.out[:n])
	m.out = m.out[n:]
	return out
}

// Pending diz quantas amostras mixadas aguardam.
func (m *Mixer) Pending() int { return len(m.out) }

// Flush devolve tudo o que sobrou, completado com silêncio até um múltiplo de n.
func (m *Mixer) Flush(n int) []int16 {
	for len(m.op) > 0 || len(m.peer) > 0 {
		m.Tick()
	}
	if rem := len(m.out) % n; rem != 0 {
		m.out = append(m.out, make([]int16, n-rem)...)
	}
	out := m.out
	m.out = nil
	return out
}

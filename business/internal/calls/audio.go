package calls

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// FrameBytes é um quadro de 20 ms de PCM 16 kHz mono Int16 LE.
	FrameBytes = 640
	// bridgeQueue limita cada sentido a 1 s de áudio. Cheia, o quadro mais antigo
	// é descartado: atraso acumulado é pior que um pico de ruído, e um destino
	// lento nunca pode travar quem produz.
	bridgeQueue = 50
	// DropGrace é por quanto tempo o navegador pode ficar sem canal de áudio antes
	// de a chamada ser encerrada.
	DropGrace = 10 * time.Second
)

var ErrAudioBusy = errors.New("a chamada já tem um canal de áudio aberto")

// Pipe é uma fila limitada de quadros: Push nunca bloqueia (descarta o mais
// antigo) e Out entrega na ordem de chegada.
type Pipe struct {
	ch      chan []byte
	dropped atomic.Int64
}

func newPipe() *Pipe { return &Pipe{ch: make(chan []byte, bridgeQueue)} }

// Push enfileira um quadro sem nunca bloquear.
func (p *Pipe) Push(f []byte) {
	for {
		select {
		case p.ch <- f:
			return
		default:
			select {
			case <-p.ch:
				p.dropped.Add(1)
			default:
			}
		}
	}
}

func (p *Pipe) Out() <-chan []byte { return p.ch }
func (p *Pipe) Dropped() int64     { return p.dropped.Load() }
func (p *Pipe) Len() int           { return len(p.ch) }

// Bridge liga o áudio do navegador ao do engine para UMA chamada. As duas
// pontas são WebSockets; este tipo só cuida do que passa no meio.
type Bridge struct {
	CallID string
	// ToEngine leva o PCM do operador ao contato; ToBrowser leva o do contato ao operador.
	ToEngine  *Pipe
	ToBrowser *Pipe
	// Tap recebe uma cópia de cada quadro dos dois sentidos (gravação). Nunca bloqueia.
	Tap func(fromOperator bool, frame []byte)

	done chan struct{}
	once sync.Once
}

func newBridge(callID string) *Bridge {
	return &Bridge{CallID: callID, ToEngine: newPipe(), ToBrowser: newPipe(), done: make(chan struct{})}
}

// IsVideoFrame diz se a mensagem binária é um quadro de VÍDEO (prefixo FF 56 44 01 + cabeçalho), o
// mesmo formato do engine. Tudo o mais é PCM de áudio, como sempre foi.
func IsVideoFrame(msg []byte) bool {
	return len(msg) > videoHeaderLen && msg[0] == 0xFF && msg[1] == 'V' && msg[2] == 'D' && msg[3] == 0x01
}

// videoHeaderLen: prefixo (4) + flags (1) + timestamp de 90 kHz (4).
const videoHeaderLen = 9

// FromBrowser registra um quadro vindo do operador. O vídeo do navegador (envio, fase 2) ainda não é
// suportado: é descartado em vez de seguir ao engine e à gravação como se fosse áudio.
func (b *Bridge) FromBrowser(f []byte) {
	if IsVideoFrame(f) {
		return
	}
	if b.Tap != nil {
		b.Tap(true, f)
	}
	b.ToEngine.Push(f)
}

// FromEngine registra um quadro vindo do contato. O vídeo segue ao navegador pela mesma fila, mas NUNCA
// entra no Tap: o gravador é só de áudio.
func (b *Bridge) FromEngine(f []byte) {
	if b.Tap != nil && !IsVideoFrame(f) {
		b.Tap(false, f)
	}
	b.ToBrowser.Push(f)
}

func (b *Bridge) Done() <-chan struct{} { return b.done }
func (b *Bridge) Close()                { b.once.Do(func() { close(b.done) }) }

// Audio guarda as pontes abertas e aplica as regras de acesso (um canal por
// chamada, só o operador que assumiu).
type Audio struct {
	mu      sync.Mutex
	bridges map[string]*Bridge
	dropTO  time.Duration
}

func NewAudio() *Audio { return &Audio{bridges: map[string]*Bridge{}, dropTO: DropGrace} }

// Open abre a ponte da chamada, no máximo uma por chamada.
func (a *Audio) Open(callID string) (*Bridge, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.bridges[callID]; ok {
		return nil, ErrAudioBusy
	}
	b := newBridge(callID)
	a.bridges[callID] = b
	return b, nil
}

// Release fecha e remove a ponte.
func (a *Audio) Release(b *Bridge) {
	a.mu.Lock()
	if a.bridges[b.CallID] == b {
		delete(a.bridges, b.CallID)
	}
	a.mu.Unlock()
	b.Close()
}

// Active diz se há ponte aberta para a chamada.
func (a *Audio) Active(callID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.bridges[callID]
	return ok
}

// Count devolve quantas pontes estão abertas (carga).
func (a *Audio) Count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.bridges)
}

// Watchdog encerra a chamada se, depois que a ponte cair, ela não voltar em
// DropGrace. `ended` diz se a chamada já acabou por outro motivo; `end` a encerra.
func (a *Audio) Watchdog(ctx context.Context, callID string, ended func() bool, end func()) {
	t := time.NewTimer(a.dropTO)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
		if !ended() && !a.Active(callID) {
			end()
		}
	}
}

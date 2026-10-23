package calls

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func frame(b byte) []byte { return bytes.Repeat([]byte{b}, FrameBytes) }

func TestPipe_KeepsOrderAndNeverBlocks(t *testing.T) {
	p := newPipe()
	for i := 0; i < 10; i++ {
		p.Push(frame(byte(i)))
	}
	for i := 0; i < 10; i++ {
		assert.Equal(t, frame(byte(i)), <-p.Out(), "os quadros chegam na ordem em que entraram")
	}
}

func TestPipe_SlowConsumerIsBoundedAndProducerNeverBlocks(t *testing.T) {
	p := newPipe()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10000; i++ {
			p.Push(frame(byte(i)))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("o produtor travou por causa de um consumidor lento")
	}
	assert.LessOrEqual(t, p.Len(), bridgeQueue, "a fila nunca passa do limite")
	assert.GreaterOrEqual(t, p.Dropped(), int64(10000-bridgeQueue), "o excedente é descartado")
	last := byte((10000 - 1) % 256)
	var got []byte
	for p.Len() > 0 {
		got = <-p.Out()
	}
	assert.Equal(t, last, got[0], "sobra o áudio mais recente, não o mais antigo")
}

func TestBridge_BothDirectionsAndTap(t *testing.T) {
	a := NewAudio()
	b, err := a.Open("C1")
	require.NoError(t, err)
	var mu sync.Mutex
	var taps []bool
	b.Tap = func(fromOp bool, _ []byte) { mu.Lock(); taps = append(taps, fromOp); mu.Unlock() }

	b.FromBrowser(frame(1))
	b.FromEngine(frame(2))
	assert.Equal(t, frame(1), <-b.ToEngine.Out(), "operador → engine")
	assert.Equal(t, frame(2), <-b.ToBrowser.Out(), "engine → operador")
	mu.Lock()
	assert.Equal(t, []bool{true, false}, taps, "a gravação vê os dois sentidos")
	mu.Unlock()
}

func TestAudio_OnlyOneBridgePerCall(t *testing.T) {
	a := NewAudio()
	b, err := a.Open("C1")
	require.NoError(t, err)
	_, err = a.Open("C1")
	assert.Equal(t, ErrAudioBusy, err)
	other, err := a.Open("C2")
	require.NoError(t, err)
	assert.Equal(t, 2, a.Count())

	a.Release(b)
	assert.False(t, a.Active("C1"))
	select {
	case <-b.Done():
	default:
		t.Fatal("Release deve fechar a ponte")
	}
	_, err = a.Open("C1")
	assert.NoError(t, err, "depois de liberar, pode reabrir")
	a.Release(other)
}

func TestAudio_ReleaseOfStaleBridgeDoesNotKillNewOne(t *testing.T) {
	a := NewAudio()
	old, _ := a.Open("C1")
	a.Release(old)
	fresh, _ := a.Open("C1")
	a.Release(old)
	assert.True(t, a.Active("C1"), "liberar a ponte velha não derruba a nova")
	a.Release(fresh)
}

// 7.5: sem o canal por mais de 10 s, a chamada é encerrada; se ele volta antes, não.
func TestWatchdog_EndsCallWhenBridgeStaysDown(t *testing.T) {
	a := NewAudio()
	a.dropTO = 40 * time.Millisecond
	var ended atomic.Int32
	go a.Watchdog(context.Background(), "C1", func() bool { return false }, func() { ended.Add(1) })
	time.Sleep(150 * time.Millisecond)
	assert.EqualValues(t, 1, ended.Load(), "a ponte não voltou: encerra a chamada")
}

func TestWatchdog_DoesNothingIfBridgeComesBack(t *testing.T) {
	a := NewAudio()
	a.dropTO = 60 * time.Millisecond
	var ended atomic.Int32
	go a.Watchdog(context.Background(), "C1", func() bool { return false }, func() { ended.Add(1) })
	time.Sleep(20 * time.Millisecond)
	b, _ := a.Open("C1")
	time.Sleep(150 * time.Millisecond)
	assert.Zero(t, ended.Load(), "o operador reconectou a tempo")
	a.Release(b)
}

func TestWatchdog_DoesNothingIfCallAlreadyEnded(t *testing.T) {
	a := NewAudio()
	a.dropTO = 30 * time.Millisecond
	var ended atomic.Int32
	go a.Watchdog(context.Background(), "C1", func() bool { return true }, func() { ended.Add(1) })
	time.Sleep(120 * time.Millisecond)
	assert.Zero(t, ended.Load())
}

func TestWatchdog_StopsOnContextCancel(t *testing.T) {
	a := NewAudio()
	a.dropTO = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() { a.Watchdog(ctx, "C1", func() bool { return false }, func() {}); close(finished) }()
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("o watchdog não parou com o contexto cancelado (vazamento de goroutine)")
	}
}

// Causa do teste instável TestServeAudio_VideoNeverReachesTheAudioRecorder: a fila entre o engine e o navegador
// guarda no máximo bridgeQueue quadros e, cheia, DESCARTA o mais antigo (de propósito: atraso acumulado é pior que
// perda). Um teste que escreve MAIS que bridgeQueue quadros e depois espera ler TODOS depende de o consumidor
// esvaziar a fila a tempo: com um consumidor lento, o excedente é descartado e a leitura trava. Aqui o comportamento
// é provado sem depender de agendamento: com ninguém consumindo, só os bridgeQueue mais recentes sobram.
func TestPipe_OverCapacityKeepsOnlyTheMostRecent_NeverAllOfThem(t *testing.T) {
	p := newPipe()
	over := bridgeQueue + 10
	for i := 0; i < over; i++ {
		p.Push(frame(byte(i)))
	}
	assert.Equal(t, bridgeQueue, p.Len(), "a fila nunca guarda mais que o limite")
	assert.Equal(t, int64(over-bridgeQueue), p.Dropped(), "o excedente é descartado, não enfileirado")
	first := <-p.Out()
	assert.Equal(t, byte(over-bridgeQueue), first[0], "o primeiro que sobra é o mais antigo DENTRO da janela, não o quadro 0")
}

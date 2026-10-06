package recording

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	mp3 "github.com/alltomatos/watinkdev/business/internal/recording/shine"
)

const (
	// bitrateKbps: mono 16 kHz a 32 kbps = 14,4 MB/h, suficiente para voz.
	bitrateKbps = 32
	// mp3Block é o tamanho de bloco que o shine-mp3 consome por vez (1 quadro MPEG-2
	// a 16 kHz = 576 amostras). Alimentá-lo com outro tamanho corrompe o MP3.
	mp3Block = 576
)

var ErrClosed = errors.New("gravação já encerrada")

// Recorder grava uma chamada: recebe os dois sentidos, mixa por relógio e codifica
// em MP3 num arquivo temporário em disco (a chamada pode durar horas: nada de
// acumular o áudio inteiro em memória).
//
// Seguro para uso concorrente.
type Recorder struct {
	mu      sync.Mutex
	mixer   Mixer
	enc     *mp3.Encoder
	file    *os.File
	out     *bufio.Writer
	samples int64
	closed  bool
	errored error
}

// New cria o gravador com o arquivo temporário em dir ("" = padrão do sistema).
func New(dir string) (*Recorder, error) {
	f, err := os.CreateTemp(dir, "call-*.mp3")
	if err != nil {
		return nil, fmt.Errorf("arquivo temporário da gravação: %w", err)
	}
	return &Recorder{enc: mp3.NewEncoderBitrate(SampleRate, 1, bitrateKbps), file: f, out: bufio.NewWriterSize(f, 64*1024)}, nil
}

// AddOperator e AddPeer entregam um quadro PCM Int16 LE de cada lado. Nunca
// bloqueiam além do mutex e ignoram entradas após o fechamento.
func (r *Recorder) AddOperator(pcm []byte) { r.add(pcm, true) }
func (r *Recorder) AddPeer(pcm []byte)     { r.add(pcm, false) }

func (r *Recorder) add(pcm []byte, operator bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.errored != nil {
		return
	}
	if operator {
		r.mixer.AddOperator(pcm)
	} else {
		r.mixer.AddPeer(pcm)
	}
}

// Tick avança o relógio um quadro (20 ms): mixa e codifica o que já fecha um bloco
// de 576 amostras. O gravador de produção o chama a cada 20 ms.
func (r *Recorder) Tick() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrClosed
	}
	r.mixer.Tick()
	return r.drainLocked(false)
}

// drainLocked codifica blocos completos de 576 amostras. Com final=true completa o
// resto com silêncio até fechar um bloco.
func (r *Recorder) drainLocked(final bool) error {
	if r.errored != nil {
		return r.errored
	}
	var blocks []int16
	if final {
		blocks = r.mixer.Flush(mp3Block)
	} else {
		for r.mixer.Pending() >= mp3Block {
			blocks = append(blocks, r.mixer.Take(mp3Block)...)
		}
	}
	for i := 0; i+mp3Block <= len(blocks); i += mp3Block {
		if err := r.enc.Write(r.out, blocks[i:i+mp3Block]); err != nil {
			r.errored = err
			return err
		}
		r.samples += mp3Block
	}
	return nil
}

// Duration é o tempo de áudio já codificado.
func (r *Recorder) Duration() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return time.Duration(r.samples) * time.Second / SampleRate
}

// Finish encerra a gravação: completa o último quadro com silêncio, descarrega o
// arquivo e o devolve ABERTO para leitura desde o início, junto da duração. Quem
// chama é dono do arquivo (deve fechá-lo e removê-lo via Discard/Cleanup).
func (r *Recorder) Finish() (*os.File, time.Duration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, 0, ErrClosed
	}
	r.closed = true
	if err := r.drainLocked(true); err != nil {
		return nil, 0, err
	}
	if err := r.out.Flush(); err != nil {
		return nil, 0, err
	}
	if _, err := r.file.Seek(0, 0); err != nil {
		return nil, 0, err
	}
	return r.file, time.Duration(r.samples) * time.Second / SampleRate, nil
}

// Discard descarta a gravação: fecha e apaga o arquivo temporário. Idempotente.
func (r *Recorder) Discard() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	if r.file != nil {
		name := r.file.Name()
		_ = r.file.Close()
		_ = os.Remove(name)
		r.file = nil
	}
}

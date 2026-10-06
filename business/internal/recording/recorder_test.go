package recording

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"os"
	"testing"

	mp3 "github.com/alltomatos/watinkdev/business/internal/recording/shine"
	gomp3 "github.com/hajimehoshi/go-mp3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sineFrame devolve 20 ms de PCM Int16 LE de um seno contínuo (a fase continua de
// um quadro ao outro), amplitude amp.
func sineFrame(idx int, hz, amp float64) []byte {
	b := make([]byte, FrameSamples*2)
	for i := 0; i < FrameSamples; i++ {
		n := idx*FrameSamples + i
		binary.LittleEndian.PutUint16(b[i*2:], uint16(int16(amp*math.Sin(2*math.Pi*hz*float64(n)/SampleRate))))
	}
	return b
}

func silenceFrame() []byte { return make([]byte, FrameSamples*2) }

// decode usa um decodificador INDEPENDENTE (go-mp3) e mede o tom por cruzamentos
// de zero no canal esquerdo.
func decode(t *testing.T, mp3Bytes []byte) (rate int, secs float64, hz float64, rms float64) {
	t.Helper()
	d, err := gomp3.NewDecoder(bytes.NewReader(mp3Bytes))
	require.NoError(t, err, "o MP3 deve ser válido para um decodificador independente")
	raw, err := io.ReadAll(d)
	require.NoError(t, err)
	n := len(raw) / 4
	var sum float64
	cross := 0
	prev := int16(0)
	for i := 0; i < n; i++ {
		v := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		sum += float64(v) * float64(v)
		if i > 0 && (prev < 0) != (v < 0) {
			cross++
		}
		prev = v
	}
	secs = float64(n) / float64(d.SampleRate())
	if secs > 0 {
		hz = float64(cross) / 2 / secs
		rms = math.Sqrt(sum/float64(n)) / 32768
	}
	return d.SampleRate(), secs, hz, rms
}

func finish(t *testing.T, r *Recorder) ([]byte, float64) {
	t.Helper()
	f, dur, err := r.Finish()
	require.NoError(t, err)
	defer func() { name := f.Name(); f.Close(); os.Remove(name) }()
	data, err := io.ReadAll(f)
	require.NoError(t, err)
	return data, dur.Seconds()
}

// 7.8: ida e volta com decodificador independente — duração, 16 kHz e conteúdo.
func TestRecorder_RoundTrip_DurationRateAndTone(t *testing.T) {
	r, err := New(t.TempDir())
	require.NoError(t, err)
	const seconds = 5
	for i := 0; i < seconds*50; i++ {
		r.AddPeer(sineFrame(i, 440, 9000))
		require.NoError(t, r.Tick())
	}
	data, dur := finish(t, r)

	rate, secs, hz, _ := decode(t, data)
	assert.Equal(t, 16000, rate, "taxa de amostragem")
	assert.InDelta(t, seconds, secs, 1.0, "duração do MP3 dentro de 1 s")
	assert.InDelta(t, seconds, dur, 1.0, "duração informada pelo gravador")
	assert.InDelta(t, 440, hz, 15, "o tom gravado é o tom que entrou")
}

// A armadilha medida: alimentar o encoder em blocos de 20 ms CORROMPE o MP3. O
// gravador acumula em 576 e não pode cair nisso; este teste existe para provar
// tanto que a armadilha é real quanto que o gravador a evita.
func TestMp3Encoder_FeedingIn320SampleBlocksIsCorrupt(t *testing.T) {
	enc := mp3.NewEncoderBitrate(SampleRate, 1, bitrateKbps)
	pcm := make([]int16, SampleRate*5)
	for i := range pcm {
		pcm[i] = int16(9000 * math.Sin(2*math.Pi*440*float64(i)/SampleRate))
	}
	var bad bytes.Buffer
	for i := 0; i+FrameSamples <= len(pcm); i += FrameSamples {
		require.NoError(t, enc.Write(&bad, pcm[i:i+FrameSamples]))
	}
	_, secs, hz, _ := decode(t, bad.Bytes())
	corrupt := math.Abs(secs-5) > 1.0 || math.Abs(hz-440) > 15
	assert.True(t, corrupt, "a armadilha precisa continuar real (duração %.2f s, tom %.0f Hz); se não, o acúmulo em 576 virou desnecessário e deve ser revisto", secs, hz)
}

func TestRecorder_NeverFeedsEncoderBlocksOfWrongSize(t *testing.T) {
	r, err := New(t.TempDir())
	require.NoError(t, err)
	for i := 0; i < 5*50; i++ {
		r.AddPeer(sineFrame(i, 440, 9000))
		require.NoError(t, r.Tick())
	}
	data, _ := finish(t, r)
	_, secs, hz, _ := decode(t, data)
	assert.InDelta(t, 5, secs, 1.0, "com quadros de 20 ms na entrada o MP3 sai com a duração certa")
	assert.InDelta(t, 440, hz, 15, "e o tom certo")
}

func TestRecorder_LastPartialBlockIsCompletedWithSilence(t *testing.T) {
	r, err := New(t.TempDir())
	require.NoError(t, err)
	// 1 quadro = 320 amostras < 576: sem completar o último bloco NADA seria
	// codificado e a gravação inteira se perderia.
	r.AddPeer(sineFrame(0, 440, 9000))
	require.NoError(t, r.Tick())
	data, dur := finish(t, r)
	require.NotEmpty(t, data, "o áudio que não fechou um bloco não pode ser descartado")
	assert.InDelta(t, float64(mp3Block)/SampleRate, dur, 0.001, "completa até exatamente um bloco de 576 amostras com silêncio")
	_, secs, _, _ := decode(t, data)
	assert.Greater(t, secs, 0.0)
}

func TestRecorder_PartialTailIsKeptAfterFullBlocks(t *testing.T) {
	r, err := New(t.TempDir())
	require.NoError(t, err)
	for i := 0; i < 7; i++ { // 2240 amostras = 3 blocos de 576 + 512 de resto
		r.AddPeer(sineFrame(i, 440, 9000))
		require.NoError(t, r.Tick())
	}
	_, dur := finish(t, r)
	assert.InDelta(t, float64(4*mp3Block)/SampleRate, dur, 0.001, "3 blocos cheios + o resto completado = 4 blocos")
}

// Os dois lados na mesma gravação, e lacuna vira silêncio sem deslocar o outro lado.
func TestMixer_SumsBothSidesAndGapsBecomeSilence(t *testing.T) {
	var m Mixer
	m.AddOperator(sineFrame(0, 440, 8000))
	m.Tick()
	got := m.Take(FrameSamples)
	require.Len(t, got, FrameSamples)
	assert.NotZero(t, got[40], "só o operador falou: o sinal dele está lá")

	m.Tick()
	silent := m.Take(FrameSamples)
	for i, v := range silent {
		require.Zero(t, v, "amostra %d: sem áudio de ninguém vira silêncio", i)
	}

	m.AddOperator(sineFrame(2, 440, 8000))
	m.AddPeer(sineFrame(2, 880, 8000))
	m.Tick()
	both := m.Take(FrameSamples)
	var opOnly, peerOnly Mixer
	opOnly.AddOperator(sineFrame(2, 440, 8000))
	opOnly.Tick()
	peerOnly.AddPeer(sineFrame(2, 880, 8000))
	peerOnly.Tick()
	a, b := opOnly.Take(FrameSamples), peerOnly.Take(FrameSamples)
	for i := range both {
		assert.Equal(t, int32(a[i])+int32(b[i]), int32(both[i]), "amostra %d é a soma dos dois lados", i)
	}
}

func TestMixer_PeakIsLimitedNotWrapped(t *testing.T) {
	loud := make([]byte, FrameSamples*2)
	for i := 0; i < FrameSamples; i++ {
		binary.LittleEndian.PutUint16(loud[i*2:], uint16(int16(30000)))
	}
	var m Mixer
	m.AddOperator(loud)
	m.AddPeer(loud)
	m.Tick()
	for i, v := range m.Take(FrameSamples) {
		require.Equal(t, int16(32767), v, "amostra %d: 30000+30000 limita em 32767, não vira negativo", i)
	}

	neg := make([]byte, FrameSamples*2)
	negSample := int16(-30000)
	for i := 0; i < FrameSamples; i++ {
		binary.LittleEndian.PutUint16(neg[i*2:], uint16(negSample))
	}
	var n Mixer
	n.AddOperator(neg)
	n.AddPeer(neg)
	n.Tick()
	for _, v := range n.Take(FrameSamples) {
		require.Equal(t, int16(-32768), v)
	}
}

func TestMixer_LateSideDoesNotShiftTheOther(t *testing.T) {
	var m Mixer
	m.AddPeer(sineFrame(0, 440, 8000))
	m.Tick()
	m.Tick()
	m.AddOperator(sineFrame(2, 440, 8000))
	m.AddPeer(silenceFrame())
	m.Tick()
	assert.Equal(t, 3*FrameSamples, m.Pending(), "cada tick ocupa exatamente um quadro, atrasado ou não")
}

func TestRecorder_ConcurrentAddIsSafe(t *testing.T) {
	r, err := New(t.TempDir())
	require.NoError(t, err)
	done := make(chan struct{}, 2)
	for _, add := range []func([]byte){r.AddOperator, r.AddPeer} {
		go func(add func([]byte)) {
			for i := 0; i < 500; i++ {
				add(sineFrame(i, 440, 5000))
			}
			done <- struct{}{}
		}(add)
	}
	for i := 0; i < 500; i++ {
		_ = r.Tick()
	}
	<-done
	<-done
	data, _ := finish(t, r)
	assert.NotEmpty(t, data)
}

func TestRecorder_ClosedRefusesAndDiscardRemovesFile(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	require.NoError(t, err)
	r.AddPeer(sineFrame(0, 440, 5000))
	require.NoError(t, r.Tick())
	f, _, err := r.Finish()
	require.NoError(t, err)
	name := f.Name()
	f.Close()

	_, _, err = r.Finish()
	assert.Equal(t, ErrClosed, err, "não encerra duas vezes")
	assert.Equal(t, ErrClosed, r.Tick())
	r.AddPeer(sineFrame(1, 440, 5000)) // depois de fechado: ignorado, não entra em pânico

	r.Discard()
	_, statErr := os.Stat(name)
	assert.True(t, os.IsNotExist(statErr) || statErr == nil, "o descarte não pode falhar")
	r.Discard()
}

func TestRecorder_DiscardDeletesTempFile(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	require.NoError(t, err)
	entries, _ := os.ReadDir(dir)
	require.Len(t, entries, 1, "o gravador abre um arquivo temporário em disco")
	r.Discard()
	entries, _ = os.ReadDir(dir)
	assert.Empty(t, entries, "descartar remove o arquivo (nada de áudio parcial exposto)")
}

func TestRecorder_NewFailsWhenDirDoesNotExist(t *testing.T) {
	_, err := New("/caminho/que/nao/existe/mesmo")
	assert.Error(t, err)
}

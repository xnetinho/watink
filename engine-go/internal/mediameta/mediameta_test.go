package mediameta

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"os/exec"
	"path/filepath"
	"testing"
)

var ctx = context.Background()

// Os testes que usam o ffmpeg real são pulados quando ele não existe (ex.: máquina de desenvolvimento). Na imagem do
// engine e na CI do container eles rodam.
func needFFmpeg(t *testing.T) {
	t.Helper()
	if !Available() {
		t.Skip("ffmpeg/ffprobe não instalados")
	}
}

func gen(t *testing.T, name string, args ...string) []byte {
	t.Helper()
	out := filepath.Join(t.TempDir(), name)
	full := append([]string{"-v", "error", "-y"}, args...)
	full = append(full, out)
	if b, err := exec.Command("ffmpeg", full...).CombinedOutput(); err != nil {
		t.Fatalf("gerar %s: %v: %s", name, err, b)
	}
	data, err := readFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func decode(t *testing.T, b []byte) (w, h int) {
	t.Helper()
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("a miniatura não é um JPEG válido: %v", err)
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func TestImage_LargaEAltaGeramMiniaturaNaProporcaoCerta(t *testing.T) {
	needFFmpeg(t)
	wide := gen(t, "w.png", "-f", "lavfi", "-i", "color=c=red:s=800x400", "-frames:v", "1")
	m := Image(ctx, wide, "png")
	if m.Width != 800 || m.Height != 400 {
		t.Fatalf("dimensões = %dx%d, esperado 800x400", m.Width, m.Height)
	}
	w, h := decode(t, m.Thumbnail)
	if w != ThumbMaxSide || h != 50 {
		t.Fatalf("miniatura = %dx%d, esperado %dx50 (proporção 2:1)", w, h, ThumbMaxSide)
	}

	tall := gen(t, "t.jpg", "-f", "lavfi", "-i", "color=c=blue:s=300x900", "-frames:v", "1")
	m = Image(ctx, tall, "jpg")
	w, h = decode(t, m.Thumbnail)
	if h != ThumbMaxSide || w != 34 {
		t.Fatalf("miniatura = %dx%d, esperado 34x%d", w, h, ThumbMaxSide)
	}
}

func TestImage_PequenaNaoEhAmpliada(t *testing.T) {
	needFFmpeg(t)
	small := gen(t, "s.png", "-f", "lavfi", "-i", "color=c=green:s=40x30", "-frames:v", "1")
	w, h := decode(t, Image(ctx, small, "png").Thumbnail)
	if w != 40 || h != 30 {
		t.Fatalf("miniatura = %dx%d; imagem pequena não pode ser ampliada", w, h)
	}
}

func TestVideo_DuracaoDimensoesEMiniatura(t *testing.T) {
	needFFmpeg(t)
	v := gen(t, "v.mp4", "-f", "lavfi", "-i", "testsrc=duration=3:size=320x240:rate=15", "-c:v", "libx264", "-pix_fmt", "yuv420p")
	m := Video(ctx, v, "mp4")
	if m.Seconds != 3 || m.Width != 320 || m.Height != 240 {
		t.Fatalf("meta = %+v", m)
	}
	if w, h := decode(t, m.Thumbnail); w != ThumbMaxSide || (h != 75 && h != 76) {
		t.Fatalf("miniatura = %dx%d, esperado %dx75 (ou 76: o ffmpeg mantém a altura par)", w, h, ThumbMaxSide)
	}
}

func TestVideo_RotacionadoTrocaLarguraPorAltura(t *testing.T) {
	needFFmpeg(t)
	base := gen(t, "b.mp4", "-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=15", "-c:v", "libx264", "-pix_fmt", "yuv420p")
	dir := t.TempDir()
	in := filepath.Join(dir, "in.mp4")
	if err := writeFile(in, base); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "rot.mp4")
	// ffmpeg ≥ 6 grava a rotação como Display Matrix (-display_rotation, opção de ENTRADA); antes era a tag "rotate".
	if b, err := exec.Command("ffmpeg", "-v", "error", "-y", "-display_rotation", "90", "-i", in, "-c", "copy", out).CombinedOutput(); err != nil {
		t.Skipf("ffmpeg desta versão não grava rotação: %v %s", err, b)
	}
	data, _ := readFile(out)
	m := Video(ctx, data, "mp4")
	if m.Width != 240 || m.Height != 320 {
		t.Fatalf("vídeo gravado em pé apareceu como %dx%d; esperado 240x320", m.Width, m.Height)
	}
}

func TestVideo_MuitoCurtoTemDuracaoMinimaDeUmSegundoEMiniatura(t *testing.T) {
	needFFmpeg(t)
	v := gen(t, "tiny.mp4", "-f", "lavfi", "-i", "testsrc=duration=0.4:size=160x120:rate=10", "-c:v", "libx264", "-pix_fmt", "yuv420p")
	m := Video(ctx, v, "mp4")
	if m.Seconds != 1 {
		t.Fatalf("duração = %d, esperado 1 (mídia de 0,4 s nunca vira 0)", m.Seconds)
	}
	if len(m.Thumbnail) == 0 {
		t.Fatal("vídeo curto ficou sem miniatura (o quadro 0,5 s não existe)")
	}
}

func TestAudio_DuracaoEOndaComOsDoisCanaisEMono(t *testing.T) {
	needFFmpeg(t)
	for name, args := range map[string][]string{
		"estereo.wav": {"-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-ac", "2"},
		"mono.mp3":    {"-f", "lavfi", "-i", "sine=frequency=330:duration=3", "-c:a", "libmp3lame"},
	} {
		m := Audio(ctx, gen(t, name, args...), filepath.Ext(name))
		if m.Seconds < 2 || m.Seconds > 3 {
			t.Errorf("%s: duração = %d", name, m.Seconds)
		}
		if len(m.Waveform) != WaveformSamples {
			t.Fatalf("%s: onda com %d barras, esperado %d", name, len(m.Waveform), WaveformSamples)
		}
		var peak byte
		for _, b := range m.Waveform {
			if b > 100 {
				t.Fatalf("%s: barra %d fora de 0..100", name, b)
			}
			if b > peak {
				peak = b
			}
		}
		if peak != 100 {
			t.Errorf("%s: a barra mais alta deve ser 100 (normalizada), foi %d", name, peak)
		}
	}
}

func TestAudio_SilencioNaoDividePorZero(t *testing.T) {
	needFFmpeg(t)
	s := gen(t, "s.wav", "-f", "lavfi", "-i", "anullsrc=r=8000:cl=mono", "-t", "2")
	m := Audio(ctx, s, "wav")
	if len(m.Waveform) != WaveformSamples {
		t.Fatalf("onda com %d barras", len(m.Waveform))
	}
	for _, b := range m.Waveform {
		if b != 0 {
			t.Fatalf("silêncio produziu barra %d", b)
		}
	}
}

func TestToOpusVoice_ConverteWavParaOggOpus(t *testing.T) {
	needFFmpeg(t)
	wav := gen(t, "a.wav", "-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-ac", "2")
	ogg, err := ToOpusVoice(ctx, wav, "wav")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(ogg, []byte("OggS")) {
		t.Fatalf("saída não é OGG: % x", ogg[:8])
	}
	if !bytes.Contains(ogg[:200], []byte("OpusHead")) {
		t.Fatal("o OGG não carrega Opus (falta OpusHead)")
	}
	if len(ogg) >= len(wav) {
		t.Errorf("Opus (%d bytes) não ficou menor que o WAV (%d)", len(ogg), len(wav))
	}
	if m := Audio(ctx, ogg, "ogg"); m.Seconds < 2 || m.Seconds > 3 {
		t.Errorf("duração depois de converter = %d", m.Seconds)
	}
}

func TestMidiaInvalidaOuVaziaNaoFalhaNemTrava(t *testing.T) {
	needFFmpeg(t)
	for name, data := range map[string][]byte{"vazio": nil, "lixo": bytes.Repeat([]byte{0xde, 0xad}, 500)} {
		if m := Image(ctx, data, "png"); m.Thumbnail != nil || m.Width != 0 {
			t.Errorf("%s: imagem inválida produziu %+v", name, m)
		}
		if m := Video(ctx, data, "mp4"); m.Seconds != 0 || m.Thumbnail != nil {
			t.Errorf("%s: vídeo inválido produziu %+v", name, m)
		}
		if m := Audio(ctx, data, "mp3"); m.Seconds != 0 || m.Waveform != nil {
			t.Errorf("%s: áudio inválido produziu %+v", name, m)
		}
		if _, err := ToOpusVoice(ctx, data, "mp3"); err == nil {
			t.Errorf("%s: conversão de lixo não pode dar certo", name)
		}
	}
}

func TestExtensaoNuncaCarregaEntradaDoUsuario(t *testing.T) {
	for in, want := range map[string]string{"png": ".png", ".MP4": ".mp4", "": ".bin", "../../etc/passwd": ".bin", "a b": ".bin", "verylongext": ".bin", "p;ng": ".bin"} {
		if got := safeExt(in); got != want {
			t.Errorf("safeExt(%q) = %q, esperado %q", in, got, want)
		}
	}
}

func TestWaveformFromPCM_PicoPorJanelaEMinimoDeAmostras(t *testing.T) {
	// 128 amostras: janela de 2 → barra i = pico das amostras 2i e 2i+1.
	pcm := make([]byte, 256)
	put := func(i int, v int16) { pcm[2*i], pcm[2*i+1] = byte(uint16(v)), byte(uint16(v)>>8) }
	put(0, 1000)
	put(1, -4000) // |−4000| é o pico da barra 0
	put(2, 2000)  // barra 1
	w := waveformFromPCM(pcm)
	if len(w) != WaveformSamples || w[0] != 100 || w[1] != 50 {
		t.Fatalf("onda = %v...", w[:4])
	}
	if got := waveformFromPCM([]byte{1}); got != nil {
		t.Errorf("1 byte não forma amostra, veio %v", got)
	}
	if got := waveformFromPCM(nil); got != nil {
		t.Errorf("nil: %v", got)
	}
	// menos amostras que barras: não pode estourar o índice
	short := make([]byte, 2*10)
	put2 := func(i int, v int16) { short[2*i], short[2*i+1] = byte(uint16(v)), byte(uint16(v)>>8) }
	put2(3, 500)
	if w := waveformFromPCM(short); len(w) != WaveformSamples {
		t.Errorf("áudio curto: %d barras", len(w))
	}
}

func needPoppler(t *testing.T) {
	t.Helper()
	if !AvailablePDF() {
		t.Skip("poppler (pdfinfo/pdftoppm) não instalado")
	}
}

// pdfBytes monta um PDF mínimo e válido de n páginas com o tamanho dado (em pontos).
func pdfBytes(n, w, h int) []byte {
	var objs []string
	kids := ""
	for i := 0; i < n; i++ {
		kids += fmt.Sprintf("%d 0 R ", 3+i)
	}
	objs = append(objs, "<</Type/Catalog/Pages 2 0 R>>", fmt.Sprintf("<</Type/Pages/Kids[%s]/Count %d>>", kids, n))
	for i := 0; i < n; i++ {
		objs = append(objs, fmt.Sprintf("<</Type/Page/Parent 2 0 R/MediaBox[0 0 %d %d]>>", w, h))
	}
	out := []byte("%PDF-1.4\n")
	var offs []int
	for i, o := range objs {
		offs = append(offs, len(out))
		out = append(out, []byte(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", i+1, o))...)
	}
	x := len(out)
	out = append(out, []byte(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(objs)+1))...)
	for _, o := range offs {
		out = append(out, []byte(fmt.Sprintf("%010d 00000 n \n", o))...)
	}
	out = append(out, []byte(fmt.Sprintf("trailer\n<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, x))...)
	return out
}

func TestPDF_PaginasEMiniaturaEmPe(t *testing.T) {
	needPoppler(t)
	m := PDF(ctx, pdfBytes(6, 595, 842))
	if m.Pages != 6 {
		t.Fatalf("páginas = %d, esperado 6", m.Pages)
	}
	if !isJPEG(m.Thumbnail) {
		t.Fatal("miniatura não é JPEG")
	}
	if w, h := decode(t, m.Thumbnail); w != 51 || h != DocThumbMaxSide || m.ThumbW != 51 || m.ThumbH != DocThumbMaxSide {
		t.Fatalf("miniatura = %dx%d (meta %dx%d); o cliente oficial manda 51x72 para A4 em pé", w, h, m.ThumbW, m.ThumbH)
	}
}

func TestPDF_PaisagemLimitaALargura(t *testing.T) {
	needPoppler(t)
	m := PDF(ctx, pdfBytes(1, 842, 595))
	if m.ThumbW != DocThumbMaxSide || m.ThumbH != 51 {
		t.Fatalf("paisagem = %dx%d, esperado 72x51", m.ThumbW, m.ThumbH)
	}
}

func TestPDF_InvalidoOuVazioNaoFalhaNemMandaLixo(t *testing.T) {
	needPoppler(t)
	for name, data := range map[string][]byte{"vazio": nil, "lixo": []byte("%PDF-1.4\nnao eh pdf\n"), "binario": bytes.Repeat([]byte{0, 1, 2}, 400)} {
		m := PDF(ctx, data)
		if m.Pages != 0 || m.Thumbnail != nil {
			t.Errorf("%s: PDF inválido produziu %+v (nunca pode inventar páginas ou mandar uma miniatura que não é imagem)", name, m)
		}
	}
}

func TestIsJPEG(t *testing.T) {
	if !isJPEG([]byte{0xff, 0xd8, 0xff, 0xe0}) || isJPEG([]byte("Syntax Error")) || isJPEG(nil) {
		t.Fatal("isJPEG errou")
	}
}

func TestSeconds_TruncaComoOClienteOficial(t *testing.T) {
	for dur, want := range map[string]int{"21.882167": 21, "4.282697": 4, "26.464000": 26, "11.506500": 11, "3.999": 3, "0.4": 1, "1.0": 1} {
		p := probeOut{}
		p.Format.Duration = dur
		if got := seconds(p); got != want {
			t.Errorf("seconds(%s) = %d, esperado %d", dur, got, want)
		}
	}
	if seconds(probeOut{}) != 0 {
		t.Error("sem duração conhecida deve devolver 0")
	}
}

package whatsapp

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/mediameta"
)

func TestExtOf_PrefereNomeDepoisUrlDepoisMimetype(t *testing.T) {
	cases := []struct {
		p    MediaCommandPayload
		want string
	}{
		{MediaCommandPayload{FileName: "a.PNG", MediaURL: "/x/b.jpg", MimeType: "image/webp"}, "PNG"},
		{MediaCommandPayload{MediaURL: "/public/media/ab.mp3"}, "mp3"},
		{MediaCommandPayload{MimeType: "audio/ogg; codecs=opus"}, "ogg"},
		{MediaCommandPayload{}, ""},
	}
	for _, c := range cases {
		if got := extOf(c.p); got != c.want {
			t.Errorf("extOf(%+v) = %q, esperado %q", c.p, got, c.want)
		}
	}
}

func TestEnrichMedia_VoiceConverteParaOpusEMarcaMimetype(t *testing.T) {
	if !mediameta.Available() {
		t.Skip("ffmpeg não instalado")
	}
	wav, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-f", "wav", "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	out, p, meta := enrichMedia(context.Background(), wav, MediaCommandPayload{MediaType: "voice", MimeType: "audio/wav", FileName: "a.wav"})
	if !bytes.HasPrefix(out, []byte("OggS")) || p.MimeType != "audio/ogg; codecs=opus" || p.MediaType != "voice" {
		t.Fatalf("voz não virou OGG/Opus: mime=%q tipo=%q", p.MimeType, p.MediaType)
	}
	if meta.Seconds < 2 || len(meta.Waveform) != mediameta.WaveformSamples {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestEnrichMedia_VoiceQueNaoConvertePassaComoArquivoDeAudio(t *testing.T) {
	if !mediameta.Available() {
		t.Skip("ffmpeg não instalado")
	}
	lixo := bytes.Repeat([]byte{1, 2, 3}, 200)
	out, p, _ := enrichMedia(context.Background(), lixo, MediaCommandPayload{MediaType: "voice", MimeType: "audio/x", FileName: "a.bin"})
	if !bytes.Equal(out, lixo) || p.MediaType != "audio" {
		t.Fatalf("falha na conversão não pode perder a mídia nem fingir que é voz: tipo=%q", p.MediaType)
	}
}

func TestEnrichMedia_DocumentoNaoEhTocado(t *testing.T) {
	in := []byte("%PDF-1.1")
	out, p, meta := enrichMedia(context.Background(), in, MediaCommandPayload{MediaType: "document", MimeType: "application/pdf"})
	if !bytes.Equal(out, in) || p.MediaType != "document" || meta.Thumbnail != nil {
		t.Fatalf("documento alterado: %+v", meta)
	}
}

// Medido com o cliente oficial (29/out/2026): arquivo de áudio chega com waveform=0B em QUALQUER formato. A onda é da gravação.
func TestEnrichMedia_ArquivoDeAudioNuncaLevaOnda(t *testing.T) {
	if !mediameta.Available() {
		t.Skip("ffmpeg não instalado")
	}
	gen := func(codec, format string) []byte {
		b, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-c:a", codec, "-f", format, "-").Output()
		if err != nil {
			t.Fatalf("gerar %s: %v", codec, err)
		}
		return b
	}
	for _, c := range []struct {
		nome, mime, ext string
		data            []byte
	}{
		{"opus", "audio/ogg", "ogg", gen("libopus", "ogg")},
		{"mp3", "audio/mpeg", "mp3", gen("libmp3lame", "mp3")},
		{"aac", "audio/aac", "aac", gen("aac", "adts")},
	} {
		_, p, m := enrichMedia(context.Background(), c.data, MediaCommandPayload{MediaType: "audio", MimeType: c.mime, FileName: "a." + c.ext})
		if m.Waveform != nil {
			t.Errorf("%s: arquivo de áudio levou onda de %d barras (o cliente oficial manda 0)", c.nome, len(m.Waveform))
		}
		if m.Seconds != 2 && m.Seconds != 1 {
			t.Errorf("%s: duração = %d", c.nome, m.Seconds)
		}
		if c.nome == "opus" && p.MimeType != "audio/ogg; codecs=opus" {
			t.Errorf("opus: audio/ogg puro não é entregue, veio %q", p.MimeType)
		}
	}
}

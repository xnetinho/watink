package whatsapp

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/mediameta"
	"go.mau.fi/whatsmeow"
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

func TestEnrichMedia_FigurinhaPNGViraWebP512(t *testing.T) {
	if !mediameta.Available() {
		t.Skip("ffmpeg não instalado")
	}
	png, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=green:s=640x480", "-frames:v", "1", "-f", "image2pipe", "-vcodec", "png", "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	out, p, meta := enrichMedia(context.Background(), png, MediaCommandPayload{MediaType: "sticker", MimeType: "image/png", FileName: "a.png"})
	info, ok := mediameta.ParseWebP(out)
	if !ok || info.Width != 512 || info.Height != 512 || p.MimeType != "image/webp" || p.ConvertError != "" {
		t.Fatalf("PNG não virou figurinha: ok=%v %+v mime=%q err=%q", ok, info, p.MimeType, p.ConvertError)
	}
	if meta.Animated || meta.Width != 512 || meta.Height != 512 {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestEnrichMedia_FigurinhaGIFSaiAnimada(t *testing.T) {
	if !mediameta.Available() {
		t.Skip("ffmpeg não instalado")
	}
	gif, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc2=s=200x200:r=12:d=2", "-f", "gif", "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	out, _, meta := enrichMedia(context.Background(), gif, MediaCommandPayload{MediaType: "sticker", MimeType: "image/gif", FileName: "a.gif"})
	info, ok := mediameta.ParseWebP(out)
	if !ok || !info.Animated || !meta.Animated {
		t.Fatalf("GIF não virou figurinha animada: ok=%v %+v meta=%+v", ok, info, meta)
	}
	if meta.Width != 512 || meta.Height != 512 {
		t.Fatalf("a animada sai sem largura e altura (o ffmpeg não lê WebP animado): %+v", meta)
	}
}

func TestEnrichMedia_FigurinhaQueNaoConverteFalhaEmVezDeSairTorta(t *testing.T) {
	if !mediameta.Available() {
		t.Skip("ffmpeg não instalado")
	}
	lixo := bytes.Repeat([]byte{1, 2, 3}, 200)
	_, p, _ := enrichMedia(context.Background(), lixo, MediaCommandPayload{MediaType: "sticker", MimeType: "image/webp", FileName: "a.webp"})
	if p.ConvertError == "" {
		t.Fatal("figurinha que não converte tem de falhar com aviso, não sair como está")
	}
}

func TestPublishStickerSent_DevolveOWebPFinalComReplaced(t *testing.T) {
	s, calls := newTestService()
	s.publishStickerSent("t1", 7, "MSG1", []byte("WEBPDATA"), true)
	if len(*calls) != 1 {
		t.Fatalf("esperava 1 evento, veio %d", len(*calls))
	}
	c := (*calls)[0]
	if c.eventType != "message.media" || c.tenantID != "t1" || c.sessionID != 7 {
		t.Fatalf("evento errado: %+v", c)
	}
	p := c.payload
	if p["messageId"] != "MSG1" || p["mimetype"] != "image/webp" || p["replaced"] != true || p["animated"] != true || p["mediaData"] != "V0VCUERBVEE=" {
		t.Fatalf("payload errado: %+v", p)
	}
}

func TestEnrichMedia_GIFComumViraVideoMP4ComGifPlayback(t *testing.T) {
	if !mediameta.Available() {
		t.Skip("ffmpeg não instalado")
	}
	gif, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc2=s=200x200:r=12:d=2", "-f", "gif", "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	out, p, meta := enrichMedia(context.Background(), gif, MediaCommandPayload{MediaType: "image", MimeType: "image/gif", FileName: "a.gif"})
	if p.MediaType != "video" || p.MimeType != "video/mp4" || p.ConvertError != "" {
		t.Fatalf("GIF não virou vídeo: tipo=%q mime=%q err=%q", p.MediaType, p.MimeType, p.ConvertError)
	}
	if len(out) < 12 || string(out[4:8]) != "ftyp" || !meta.GIF || meta.Width != 200 || meta.Seconds != 2 {
		t.Fatalf("saída/meta errados: %d bytes, meta=%+v", len(out), meta)
	}
	msg := buildMediaMessage(p, whatsmeow.UploadResponse{}, meta)
	if v := msg.GetVideoMessage(); v == nil || v.GifPlayback == nil || !*v.GifPlayback || v.GetMimetype() != "video/mp4" {
		t.Fatalf("o VideoMessage do GIF precisa de gifPlayback: %+v", v)
	}
}

func TestEnrichMedia_FotoComumNaoViraVideo(t *testing.T) {
	if !mediameta.Available() {
		t.Skip("ffmpeg não instalado")
	}
	png, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=green:s=64x64", "-frames:v", "1", "-f", "image2pipe", "-vcodec", "png", "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	_, p, meta := enrichMedia(context.Background(), png, MediaCommandPayload{MediaType: "image", MimeType: "image/png", FileName: "a.png"})
	if p.MediaType != "image" || meta.GIF {
		t.Fatalf("imagem comum não pode virar vídeo: tipo=%q gif=%v", p.MediaType, meta.GIF)
	}
}

func TestEnrichMedia_GIFQueNaoConverteFalhaEmVezDeSerDescartadoPeloCelular(t *testing.T) {
	if !mediameta.Available() {
		t.Skip("ffmpeg não instalado")
	}
	quebrado := append([]byte("GIF89a"), bytes.Repeat([]byte{7}, 100)...)
	_, p, _ := enrichMedia(context.Background(), quebrado, MediaCommandPayload{MediaType: "image", MimeType: "image/gif", FileName: "a.gif"})
	if p.ConvertError == "" {
		t.Fatal("GIF que não converte tem de falhar com aviso, não ser enviado como image/gif (o celular o descarta)")
	}
}

func TestPublishGIFSent_DevolveOMP4FinalComReplacedETrimmed(t *testing.T) {
	s, calls := newTestService()
	s.publishGIFSent("t1", 7, "MSG1", []byte("MP4DATA"), true)
	if len(*calls) != 1 {
		t.Fatalf("esperava 1 evento, veio %d", len(*calls))
	}
	p := (*calls)[0].payload
	if (*calls)[0].eventType != "message.media" || p["mimetype"] != "video/mp4" || p["mediaType"] != "video" || p["replaced"] != true || p["gif"] != true || p["trimmed"] != true || p["messageId"] != "MSG1" {
		t.Fatalf("payload errado: %+v", p)
	}
}

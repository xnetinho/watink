package mediameta

import (
	"context"
	"testing"
)

func TestParseWebP(t *testing.T) {
	needFFmpeg(t)
	st := gen(t, "a.webp", "-f", "lavfi", "-i", "color=c=blue:s=300x200", "-frames:v", "1", "-c:v", "libwebp")
	if info, ok := ParseWebP(st); !ok || info.Width != 300 || info.Height != 200 || info.Animated {
		t.Fatalf("estático: %+v ok=%v", info, ok)
	}
	an := gen(t, "b.webp", "-f", "lavfi", "-i", "testsrc2=s=64x48:r=10:d=1", "-c:v", "libwebp_anim", "-loop", "0")
	if info, ok := ParseWebP(an); !ok || info.Width != 64 || info.Height != 48 || !info.Animated {
		t.Fatalf("animado: %+v ok=%v", info, ok)
	}
	for _, bad := range [][]byte{nil, []byte("RIFF"), []byte("não é webp, só texto comprido o bastante"), {0xff, 0xd8, 0xff, 0xe0}} {
		if _, ok := ParseWebP(bad); ok {
			t.Fatalf("%q não é WebP", bad)
		}
	}
}

func TestToSticker_ImagemEstaticaViraWebP512ComFundoTransparente(t *testing.T) {
	needFFmpeg(t)
	png := gen(t, "in.png", "-f", "lavfi", "-i", "color=c=green:s=640x480", "-frames:v", "1")
	s, err := ToSticker(context.Background(), png, "png")
	if err != nil {
		t.Fatal(err)
	}
	info, ok := ParseWebP(s.Data)
	if !ok || info.Width != 512 || info.Height != 512 || info.Animated || s.Animated {
		t.Fatalf("esperava WebP estático 512x512, veio %+v animado=%v", info, s.Animated)
	}
	if len(s.Data) > MaxStaticSticker {
		t.Fatalf("estática passou de 100 KB: %d", len(s.Data))
	}
}

func TestToSticker_WebPJaCertoNaoERecodificado(t *testing.T) {
	needFFmpeg(t)
	ok := gen(t, "ok.webp", "-f", "lavfi", "-i", "color=c=red:s=512x512", "-frames:v", "1", "-c:v", "libwebp", "-lossless", "1")
	s, err := ToSticker(context.Background(), ok, "webp")
	if err != nil {
		t.Fatal(err)
	}
	if string(s.Data) != string(ok) {
		t.Fatal("um WebP 512x512 dentro do limite deve seguir byte a byte, sem recodificar")
	}
}

func TestToSticker_GIFeVideoViramAnimadaDentroDoLimite(t *testing.T) {
	needFFmpeg(t)
	gif := gen(t, "in.gif", "-f", "lavfi", "-i", "testsrc2=s=200x200:r=12:d=3")
	s, err := ToSticker(context.Background(), gif, "gif")
	if err != nil {
		t.Fatal(err)
	}
	info, ok := ParseWebP(s.Data)
	if !ok || !info.Animated || !s.Animated || info.Width != 512 || info.Height != 512 || len(s.Data) > MaxAnimatedSticker {
		t.Fatalf("GIF: %+v animado=%v bytes=%d", info, s.Animated, len(s.Data))
	}

	// vídeo mais longo que o limite: cortado em 10 s e ainda cabendo em 500 KB pela escada de qualidade
	mp4 := gen(t, "long.mp4", "-f", "lavfi", "-i", "testsrc2=s=640x360:r=25:d=12", "-c:v", "libx264", "-pix_fmt", "yuv420p")
	s, err = ToSticker(context.Background(), mp4, "mp4")
	if err != nil {
		t.Fatal(err)
	}
	if info, ok := ParseWebP(s.Data); !ok || !info.Animated || len(s.Data) > MaxAnimatedSticker {
		t.Fatalf("vídeo: %+v bytes=%d", info, len(s.Data))
	}
}

func TestToSticker_RecusaLixo(t *testing.T) {
	needFFmpeg(t)
	if _, err := ToSticker(context.Background(), []byte("isto não é imagem nenhuma"), "png"); err == nil {
		t.Fatal("arquivo que não é mídia deve ser recusado")
	}
	if _, err := ToSticker(context.Background(), nil, "png"); err == nil {
		t.Fatal("vazio deve ser recusado")
	}
}

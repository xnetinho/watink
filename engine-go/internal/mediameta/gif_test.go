package mediameta

import (
	"bytes"
	"context"
	"testing"
)

func TestIsGIF(t *testing.T) {
	if !IsGIF([]byte("GIF89a....")) || !IsGIF([]byte("GIF87a....")) {
		t.Fatal("cabeçalhos GIF não reconhecidos")
	}
	for _, no := range [][]byte{nil, []byte("GIF"), []byte("\x89PNG\r\n"), []byte("RIFF....WEBP"), []byte("gif89a")} {
		if IsGIF(no) {
			t.Fatalf("%q não é GIF", no)
		}
	}
}

func TestGIFToVideo_ViraMP4CurtoSemAudio(t *testing.T) {
	needFFmpeg(t)
	gif := gen(t, "in.gif", "-f", "lavfi", "-i", "testsrc2=s=240x240:r=12:d=3")
	v, err := GIFToVideo(context.Background(), gif)
	if err != nil {
		t.Fatal(err)
	}
	if v.Trimmed {
		t.Fatal("GIF de 3 s não é cortado")
	}
	if len(v.Data) < 12 || string(v.Data[4:8]) != "ftyp" {
		t.Fatalf("não é MP4 (sem ftyp): %q", v.Data[:min(12, len(v.Data))])
	}
	m := Video(context.Background(), v.Data, "mp4")
	if m.Width != 240 || m.Height != 240 || m.Seconds != 3 || len(m.Thumbnail) == 0 {
		t.Fatalf("o MP4 gerado não é lido como vídeo de 3 s 240x240: %+v", m)
	}
	if bytes.Contains(v.Data[:min(4096, len(v.Data))], []byte("soun")) {
		t.Fatal("o MP4 de um GIF não pode ter trilha de áudio")
	}
}

func TestGIFToVideo_DimensaoImparFicaPar(t *testing.T) {
	needFFmpeg(t)
	gif := gen(t, "odd.gif", "-f", "lavfi", "-i", "testsrc2=s=201x151:r=10:d=1")
	v, err := GIFToVideo(context.Background(), gif)
	if err != nil {
		t.Fatalf("o libx264 recusa dimensão ímpar; a conversão deve corrigir: %v", err)
	}
	m := Video(context.Background(), v.Data, "mp4")
	if m.Width%2 != 0 || m.Height%2 != 0 || m.Width == 0 {
		t.Fatalf("dimensões ímpares ou vazias: %dx%d", m.Width, m.Height)
	}
}

func TestGIFToVideo_GrandeEhReduzidoELongoEhCortado(t *testing.T) {
	needFFmpeg(t)
	gif := gen(t, "big.gif", "-f", "lavfi", "-i", "testsrc2=s=800x600:r=10:d=20")
	v, err := GIFToVideo(context.Background(), gif)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Trimmed {
		t.Fatal("GIF de 20 s deve ser marcado como cortado")
	}
	m := Video(context.Background(), v.Data, "mp4")
	if m.Width != 512 || m.Height != 384 {
		t.Fatalf("esperava 512x384 (lado maior reduzido, proporção 4:3 mantida): %dx%d", m.Width, m.Height)
	}
	if m.Seconds > GIFMaxSeconds+1 || m.Seconds < GIFMaxSeconds-1 {
		t.Fatalf("duração %ds fora do corte de %ds", m.Seconds, GIFMaxSeconds)
	}
}

func TestGIFToVideo_NaoAmplia(t *testing.T) {
	needFFmpeg(t)
	gif := gen(t, "small.gif", "-f", "lavfi", "-i", "testsrc2=s=100x80:r=10:d=1")
	v, err := GIFToVideo(context.Background(), gif)
	if err != nil {
		t.Fatal(err)
	}
	if m := Video(context.Background(), v.Data, "mp4"); m.Width != 100 || m.Height != 80 {
		t.Fatalf("GIF pequeno não deve ser ampliado: %dx%d", m.Width, m.Height)
	}
}

func TestGIFToVideo_RecusaOQueNaoEGIF(t *testing.T) {
	needFFmpeg(t)
	for _, in := range [][]byte{nil, []byte("isto não é um gif"), []byte("GIF89a\x00\x00lixo")} {
		if _, err := GIFToVideo(context.Background(), in); err == nil {
			t.Fatalf("%q deveria ser recusado", in)
		}
	}
}

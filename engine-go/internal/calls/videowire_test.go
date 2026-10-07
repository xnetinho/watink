package calls

import (
	"bytes"
	"testing"
)

func TestVideoFrameRoundTrip(t *testing.T) {
	au := []byte{0, 0, 0, 1, 0x65, 1, 2, 3}
	for _, key := range []bool{true, false} {
		msg := EncodeVideoFrame(0xDEADBEEF, key, au)
		if !IsVideoFrame(msg) {
			t.Fatal("quadro de vídeo não reconhecido")
		}
		ts, k, got, err := DecodeVideoFrame(msg)
		if err != nil || ts != 0xDEADBEEF || k != key || !bytes.Equal(got, au) {
			t.Fatalf("round-trip: ts=%x key=%v au=%x err=%v", ts, k, got, err)
		}
	}
}

// O áudio PCM (640 bytes) nunca pode ser lido como vídeo, nem por acaso nem por um sinal que comece com
// o mesmo prefixo: um PCM de 640 bytes tem tamanho fixo e o vídeo exige o prefixo mágico.
func TestPcmIsNeverMistakenForVideo(t *testing.T) {
	for _, pcm := range [][]byte{
		make([]byte, 640),
		bytes.Repeat([]byte{0xFF}, 640),
		bytes.Repeat([]byte{0x00, 0x80}, 320),
		{},
		{0xFF},
		{0xFF, 'V', 'D'},
		{0xFF, 'V', 'D', 0x01}, // só o prefixo, sem corpo
	} {
		if IsVideoFrame(pcm) {
			t.Fatalf("%x... foi lido como vídeo", pcm[:min(len(pcm), 8)])
		}
	}
	if _, _, _, err := DecodeVideoFrame(make([]byte, 640)); err != ErrNotVideoFrame {
		t.Fatalf("PCM decodificado como vídeo: %v", err)
	}
}

func TestVideoFrameRejectsTruncated(t *testing.T) {
	msg := EncodeVideoFrame(1, true, []byte{1})
	if IsVideoFrame(msg[:videoHeaderLen]) {
		t.Fatal("sem corpo não é um quadro")
	}
}

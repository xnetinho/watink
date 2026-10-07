package calls

import (
	"bytes"
	"testing"
)

func TestVideoFrameRoundTrip(t *testing.T) {
	au := []byte{0, 0, 0, 1, 0x65, 1, 2, 3}
	for _, key := range []bool{true, false} {
		msg := EncodeVideoFrame(0xDEADBEEF, key, 0, au)
		if !IsVideoFrame(msg) {
			t.Fatal("quadro de vídeo não reconhecido")
		}
		ts, k, _, got, err := DecodeVideoFrame(msg)
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
	if _, _, _, _, err := DecodeVideoFrame(make([]byte, 640)); err != ErrNotVideoFrame {
		t.Fatalf("PCM decodificado como vídeo: %v", err)
	}
}

func TestVideoFrameRejectsTruncated(t *testing.T) {
	msg := EncodeVideoFrame(1, true, 0, []byte{1})
	if IsVideoFrame(msg[:videoHeaderLen]) {
		t.Fatal("sem corpo não é um quadro")
	}
}

// A rotação (0..3 quartos de volta) viaja nos bits 1-2 de flags, ao lado do bit de quadro-chave.
func TestVideoFrameCarriesRotation(t *testing.T) {
	au := []byte{0, 0, 0, 1, 0x65, 1}
	for rot := 0; rot <= 3; rot++ {
		for _, key := range []bool{true, false} {
			msg := EncodeVideoFrame(7, key, rot, au)
			_, k, r, body, err := DecodeVideoFrame(msg)
			if err != nil || k != key || r != rot || !bytes.Equal(body, au) {
				t.Fatalf("rot=%d key=%v: k=%v r=%d err=%v", rot, key, k, r, err)
			}
		}
	}
}

// Valor fora de 0..3 nunca pode vazar para o bit de quadro-chave nem estourar os 2 bits.
func TestVideoFrameRotationIsClamped(t *testing.T) {
	for _, rot := range []int{-1, 4, 5, 99} {
		msg := EncodeVideoFrame(1, false, rot, []byte{1})
		_, key, r, _, _ := DecodeVideoFrame(msg)
		if key || r != 0 {
			t.Fatalf("rot=%d virou key=%v r=%d: valor inválido deve ser 0 e não pode ligar o quadro-chave", rot, key, r)
		}
	}
}

// Vetor fixo: o navegador (frontend) usa os MESMOS bytes, para os dois lados nunca divergirem.
func TestVideoFrameWireVector(t *testing.T) {
	msg := EncodeVideoFrame(0x01020304, true, 3, []byte{0xaa, 0xbb})
	want := []byte{0xff, 0x56, 0x44, 0x01, 0x07, 0x01, 0x02, 0x03, 0x04, 0xaa, 0xbb}
	if !bytes.Equal(msg, want) {
		t.Fatalf("bytes do quadro\n got  %x\n want %x", msg, want)
	}
}

// Bits altos de flags (reservados) não podem vazar para a rotação: só os bits 1-2 contam.
func TestVideoFrameIgnoresReservedFlagBits(t *testing.T) {
	msg := EncodeVideoFrame(1, false, 0, []byte{1})
	msg[4] |= 0xF0 // lixo nos bits reservados
	_, key, rot, _, err := DecodeVideoFrame(msg)
	if err != nil || key || rot != 0 {
		t.Fatalf("bits reservados vazaram: key=%v rot=%d err=%v", key, rot, err)
	}
}

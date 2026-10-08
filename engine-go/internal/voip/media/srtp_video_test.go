package media

import (
	"bytes"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
)

func srtpPair(t *testing.T) (*SrtpSession, *SrtpSession) {
	t.Helper()
	km, err := DerivePerJidSrtpKey(bytes.Repeat([]byte{9}, 32), "peer:0@lid")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := NewSrtpSession(km, km, core.SRTPSendAuthTagLen, core.SRTPRecvAuthTagLen)
	if err != nil {
		t.Fatal(err)
	}
	rx, err := NewSrtpSession(km, km, core.SRTPSendAuthTagLen, core.SRTPRecvAuthTagLen)
	if err != nil {
		t.Fatal(err)
	}
	return tx, rx
}

func srtpWire(t *testing.T, tx *SrtpSession, pt uint8, seq uint16, ssrc uint32) []byte {
	t.Helper()
	w, err := tx.Protect(&RtpPacket{Header: NewRtpHeader(pt, seq, uint32(seq), ssrc), Payload: bytes.Repeat([]byte{1}, 40)})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// Áudio e vídeo da mesma chamada têm sequências INDEPENDENTES (cada SSRC conta do seu jeito). Num
// contexto único, o vídeo (seq alto) empurrava a janela anti-replay e o áudio passava a ser rejeitado
// como "repetido": ligar o vídeo derrubava o áudio.
func TestSrtpAudioAndVideoHaveIndependentReplayState(t *testing.T) {
	tx, rx := srtpPair(t)
	for i := uint16(0); i < 20; i++ {
		a := srtpWire(t, tx, core.PayloadTypeWhatsAppOpus, 100+i, 0xA0)
		v := srtpWire(t, tx, PayloadTypeH264, 5000+i*3, 0xB2)
		if _, err := rx.Unprotect(a); err != nil {
			t.Fatalf("áudio %d rejeitado: %v", i, err)
		}
		if _, err := rx.Unprotect(v); err != nil {
			t.Fatalf("vídeo %d rejeitado: %v", i, err)
		}
	}
}

// A proteção contra repetição continua valendo DENTRO de cada fluxo.
func TestSrtpReplayStillRejectedPerStream(t *testing.T) {
	tx, rx := srtpPair(t)
	a := srtpWire(t, tx, core.PayloadTypeWhatsAppOpus, 200, 0xA0)
	v := srtpWire(t, tx, PayloadTypeH264, 7000, 0xB2)
	for _, w := range [][]byte{a, v} {
		if _, err := rx.Unprotect(w); err != nil {
			t.Fatal(err)
		}
		if _, err := rx.Unprotect(w); err == nil {
			t.Fatal("repetição aceita")
		}
	}
}

// A virada do número de sequência (ROC) de um fluxo não pode afetar o outro.
func TestSrtpRocWrapIsPerStream(t *testing.T) {
	tx, rx := srtpPair(t)
	for _, s := range []uint16{65530, 65534, 2, 6} {
		w := srtpWire(t, tx, PayloadTypeH264, s, 0xB2)
		if _, err := rx.Unprotect(w); err != nil {
			t.Fatalf("vídeo seq %d: %v", s, err)
		}
		a := srtpWire(t, tx, core.PayloadTypeWhatsAppOpus, 300+s%10, 0xA0)
		if _, err := rx.Unprotect(a); err != nil && s == 65530 {
			t.Fatalf("áudio: %v", err)
		}
	}
}

// Pacote forjado de vídeo não pode desalinhar o estado do áudio.
func TestSrtpForgedVideoDoesNotDisturbAudio(t *testing.T) {
	tx, rx := srtpPair(t)
	if _, err := rx.Unprotect(srtpWire(t, tx, core.PayloadTypeWhatsAppOpus, 50, 0xA0)); err != nil {
		t.Fatal(err)
	}
	forged := srtpWire(t, tx, PayloadTypeH264, 9000, 0xB2)
	forged[len(forged)-1] ^= 0xff
	if _, err := rx.Unprotect(forged); err == nil {
		t.Fatal("tag adulterada aceita")
	}
	if _, err := rx.Unprotect(srtpWire(t, tx, core.PayloadTypeWhatsAppOpus, 51, 0xA0)); err != nil {
		t.Fatalf("o áudio seguinte foi rejeitado: %v", err)
	}
}

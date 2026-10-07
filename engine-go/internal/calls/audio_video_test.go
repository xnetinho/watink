package calls

import (
	"bytes"
	"context"
	"testing"
	"time"
)

// A fila de vídeo é separada da de áudio: um vídeo pesado nunca pode atrasar nem empurrar o PCM.
func TestVideoPipe_IsSeparateFromAudioQueue(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	_ = r.s.Accept(context.Background(), callA)
	p, err := r.s.OpenAudio(callA)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	eventually(t, "mídia ligada", func() bool { return r.handle(0).media.OnPeerVideo != nil })

	for i := 0; i < videoQueueFrames+10; i++ {
		r.handle(0).media.OnPeerVideo([]byte{0, 0, 0, 1, 0x41, byte(i)}, false)
	}
	select {
	case <-p.Out():
		t.Fatal("o vídeo não pode aparecer na fila de áudio")
	default:
	}
	if got := len(p.Video()); got != videoQueueFrames {
		t.Fatalf("fila de vídeo com %d itens, esperado o limite %d (descarta o mais antigo)", got, videoQueueFrames)
	}
	// o descarte é do MAIS ANTIGO: o último enviado tem de estar na fila
	var last VideoFrame
	for len(p.Video()) > 0 {
		last = <-p.Video()
	}
	if want := byte(videoQueueFrames + 9); last.AccessUnit[len(last.AccessUnit)-1] != want {
		t.Fatalf("o último quadro devia ser %d, veio %d", want, last.AccessUnit[len(last.AccessUnit)-1])
	}
}

func TestVideoPipe_KeyframeFlagAndCopy(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	_ = r.s.Accept(context.Background(), callA)
	p, _ := r.s.OpenAudio(callA)
	defer p.Close()
	eventually(t, "mídia ligada", func() bool { return r.handle(0).media.OnPeerVideo != nil })

	buf := []byte{0, 0, 0, 1, 0x65, 1, 2, 3}
	r.handle(0).media.OnPeerVideo(buf, true)
	buf[5] = 99 // o chamador reutiliza o buffer: a cópia da fila não pode mudar
	select {
	case f := <-p.Video():
		if !f.Keyframe {
			t.Fatal("flag de keyframe perdida")
		}
		if !bytes.Equal(f.AccessUnit, []byte{0, 0, 0, 1, 0x65, 1, 2, 3}) {
			t.Fatalf("a fila guardou uma referência, não uma cópia: %x", f.AccessUnit)
		}
	case <-time.After(time.Second):
		t.Fatal("quadro não chegou")
	}
}

// Sem canal aberto o vídeo é descartado (não acumula, não bloqueia o relay).
func TestVideoPipe_NoPipeDropsSilently(t *testing.T) {
	r := newRig(t, false, nil)
	r.s.OnOffer(context.Background(), offer(callA, pn("5511999990001")))
	_ = r.s.Ready(context.Background(), callA)
	_ = r.s.Accept(context.Background(), callA)
	eventually(t, "mídia ligada", func() bool { return r.handle(0).media.OnPeerVideo != nil })
	done := make(chan struct{})
	go func() { r.handle(0).media.OnPeerVideo([]byte{1, 2, 3}, false); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("OnPeerVideo bloqueou sem canal aberto")
	}
}

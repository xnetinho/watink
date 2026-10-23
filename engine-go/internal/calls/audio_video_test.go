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
	eventually(t, "mídia ligada", func() bool { return r.handle(0).media.OnPeerVideoFrame != nil })

	for i := 0; i < videoQueueFrames+10; i++ {
		r.handle(0).media.OnPeerVideoFrame([]byte{0, 0, 0, 1, 0x41, byte(i)}, false, 0)
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
	eventually(t, "mídia ligada", func() bool { return r.handle(0).media.OnPeerVideoFrame != nil })

	buf := []byte{0, 0, 0, 1, 0x65, 1, 2, 3}
	r.handle(0).media.OnPeerVideoFrame(buf, true, 2)
	buf[5] = 99 // o chamador reutiliza o buffer: a cópia da fila não pode mudar
	select {
	case f := <-p.Video():
		if !f.Keyframe {
			t.Fatal("flag de keyframe perdida")
		}
		if f.Rotation != 2 {
			t.Fatalf("a rotação se perdeu na fila: %d", f.Rotation)
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
	eventually(t, "mídia ligada", func() bool { return r.handle(0).media.OnPeerVideoFrame != nil })
	done := make(chan struct{})
	go func() { r.handle(0).media.OnPeerVideoFrame([]byte{1, 2, 3}, false, 0); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("OnPeerVideo bloqueou sem canal aberto")
	}
}

// A duração de cada quadro da câmera sai da diferença dos timestamps de 90 kHz que o navegador manda; o
// primeiro quadro, uma diferença absurda e um quadro vazio caem no padrão (duração 0).
func TestWriteVideo_DerivesFrameDurationFromTimestamps(t *testing.T) {
	r, h := answered(t)
	p, _ := r.s.OpenAudio(callA)
	au := []byte{0, 0, 0, 1, 0x65, 1}
	p.WriteVideo(EncodeVideoFrame(90000, true, 0, au))
	p.WriteVideo(EncodeVideoFrame(90000+6000, false, 0, au))
	p.WriteVideo(EncodeVideoFrame(90000+6000+3000, false, 0, au))
	p.WriteVideo(EncodeVideoFrame(90000+6000+3000+900000, false, 0, au))
	p.WriteVideo(EncodeVideoFrame(1, false, 0, nil))
	p.WriteVideo([]byte("lixo que nao e video"))

	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.videoIn) != 4 {
		t.Fatalf("chegaram %d quadros ao codec, quer 4 (vazio e lixo são ignorados)", len(h.videoIn))
	}
	want := []time.Duration{0, 6000 * time.Second / 90000, 3000 * time.Second / 90000, 0}
	for i, w := range want {
		if h.videoIn[i].d != w {
			t.Errorf("quadro %d: duração %v, quer %v", i, h.videoIn[i].d, w)
		}
	}
}

func TestSetCamera_OffForgetsTimestampsAndOnPassesOrientation(t *testing.T) {
	r, h := answered(t)
	p, _ := r.s.OpenAudio(callA)
	au := []byte{0, 0, 0, 1, 0x65, 1}
	p.WriteVideo(EncodeVideoFrame(1000, true, 0, au))
	if err := p.SetCamera(context.Background(), false, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.SetCamera(context.Background(), true, 3); err != nil {
		t.Fatal(err)
	}
	p.WriteVideo(EncodeVideoFrame(1000+6000, true, 0, au))
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.camera) != 2 || h.camera[0] != (cameraCmd{false, 0}) || h.camera[1] != (cameraCmd{true, 3}) {
		t.Fatalf("comandos = %v", h.camera)
	}
	if h.videoIn[1].d != 0 {
		t.Fatalf("depois de desligar a câmera o relógio recomeça: duração %v, quer 0", h.videoIn[1].d)
	}
}

func TestKeyframeRequest_ReachesPipeAndDoesNotBlock(t *testing.T) {
	r, h := answered(t)
	p, _ := r.s.OpenAudio(callA)
	eventually(t, "mídia ligada", func() bool { return h.media.OnKeyframeRequested != nil })
	for i := 0; i < 5; i++ {
		h.media.OnKeyframeRequested()
	}
	select {
	case c := <-p.Control():
		if c.Type != "keyframe" {
			t.Fatalf("controle = %q", c.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("o pedido de keyframe não chegou ao canal")
	}
	if len(p.Control()) > 1 {
		t.Fatal("pedidos repetidos precisam se fundir (a fila é de 1)")
	}
}

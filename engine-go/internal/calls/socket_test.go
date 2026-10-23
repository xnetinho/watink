package calls

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	waBinary "go.mau.fi/whatsmeow/binary"
)

type fakeWire struct {
	mu        sync.Mutex
	sendErr   error
	sendBlock chan struct{}
	waiters   map[string]chan *waBinary.Node
	cancelled []string
	sent      []waBinary.Node
}

func newFakeWire() *fakeWire { return &fakeWire{waiters: map[string]chan *waBinary.Node{}} }

func (f *fakeWire) SendNode(ctx context.Context, n waBinary.Node) error {
	if f.sendBlock != nil {
		select {
		case <-f.sendBlock:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, n)
	return f.sendErr
}

func (f *fakeWire) WaitResponse(id string) chan *waBinary.Node {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan *waBinary.Node, 1)
	f.waiters[id] = ch
	return ch
}

func (f *fakeWire) CancelResponse(id string, _ chan *waBinary.Node) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancelled = append(f.cancelled, id)
	delete(f.waiters, id)
}

func (f *fakeWire) reply(id string, n *waBinary.Node) {
	f.mu.Lock()
	ch := f.waiters[id]
	f.mu.Unlock()
	ch <- n
}

func newTestSocket(w wire, timeout time.Duration) *Socket {
	return &Socket{w: w, queryTimeout: timeout}
}

func node(id string) waBinary.Node {
	return waBinary.Node{Tag: "call", Attrs: waBinary.Attrs{"id": id}}
}

func TestQuery_ReturnsResponse(t *testing.T) {
	w := newFakeWire()
	s := newTestSocket(w, time.Second)
	go func() {
		time.Sleep(20 * time.Millisecond)
		w.reply("a1", &waBinary.Node{Tag: "ack"})
	}()
	resp, err := s.Query(context.Background(), node("a1"))
	if err != nil || resp == nil || resp.Tag != "ack" {
		t.Fatalf("resp=%v err=%v", resp, err)
	}
}

func TestQuery_TimeoutReturnsNilAndCancelsWaiter(t *testing.T) {
	w := newFakeWire()
	s := newTestSocket(w, 50*time.Millisecond)
	start := time.Now()
	resp, err := s.Query(context.Background(), node("a2"))
	if resp != nil || err != nil {
		t.Fatalf("timeout deve devolver (nil,nil), veio resp=%v err=%v", resp, err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("não respeitou o timeout")
	}
	if len(w.cancelled) != 1 || w.cancelled[0] != "a2" {
		t.Fatalf("waiter não foi cancelado: %v", w.cancelled)
	}
}

func TestQuery_ContextCancelReturnsErrorAndCancelsWaiter(t *testing.T) {
	w := newFakeWire()
	s := newTestSocket(w, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(30 * time.Millisecond); cancel() }()
	start := time.Now()
	_, err := s.Query(ctx, node("a3"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("não respeitou o cancelamento")
	}
	if len(w.cancelled) != 1 {
		t.Fatalf("waiter não foi cancelado: %v", w.cancelled)
	}
}

func TestQuery_SendErrorCancelsWaiter(t *testing.T) {
	w := newFakeWire()
	w.sendErr = errors.New("socket fechado")
	s := newTestSocket(w, time.Second)
	if _, err := s.Query(context.Background(), node("a4")); err == nil {
		t.Fatal("esperava erro de envio")
	}
	if len(w.cancelled) != 1 {
		t.Fatalf("waiter vazou: %v", w.cancelled)
	}
}

func TestQuery_WithoutIDJustSends(t *testing.T) {
	w := newFakeWire()
	s := newTestSocket(w, time.Second)
	resp, err := s.Query(context.Background(), waBinary.Node{Tag: "call"})
	if resp != nil || err != nil || len(w.sent) != 1 {
		t.Fatalf("resp=%v err=%v sent=%d", resp, err, len(w.sent))
	}
}

func TestSendNode_RespectsContext(t *testing.T) {
	w := newFakeWire()
	s := newTestSocket(w, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.SendNode(ctx, node("x")); !errors.Is(err, context.Canceled) {
		t.Fatalf("ctx já cancelado deve falhar sem enviar, err=%v", err)
	}
	if len(w.sent) != 0 {
		t.Fatal("enviou com ctx cancelado")
	}

	w.sendBlock = make(chan struct{})
	ctx2, cancel2 := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel2()
	if err := s.SendNode(ctx2, node("y")); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("envio bloqueado deve respeitar o deadline, err=%v", err)
	}
}

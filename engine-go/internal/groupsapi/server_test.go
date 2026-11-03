package groupsapi

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestStart_Listens verifies the server binds the configured port and shuts down cleanly on ctx cancel.
func TestStart_Listens(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve a free port: %v", err)
	}
	_, port, _ := net.SplitHostPort(lis.Addr().String())
	_ = lis.Close()

	t.Setenv("GROUPS_API_PORT", port)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		Start(ctx, &fakeBackend{})
		close(done)
	}()

	deadline := time.Now().Add(3 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", "127.0.0.1:"+port)
		if err == nil {
			_ = conn.Close()
			lastErr = nil
			break
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("server did not start listening on :%s: %v", port, lastErr)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Start did not shut down after ctx cancel")
	}
}

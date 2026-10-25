package call

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/types"
)

func TestCallManager_AsyncHooks_OrderPreserved(t *testing.T) {
	sock := &fakeSock{}
	m := NewCallManager(sock, slog.Default())
	m.DeferPreaccept = true
	defer m.Close()

	var eventsMu sync.Mutex
	var events []string

	done := make(chan struct{})

	m.OnIncoming = func(info *CallInfo) {
		eventsMu.Lock()
		events = append(events, "incoming:"+info.CallID)
		eventsMu.Unlock()
	}

	m.OnStateChange = func(info *CallInfo) {
		eventsMu.Lock()
		events = append(events, "state:"+string(info.StateData.State))
		eventsMu.Unlock()
	}

	m.OnEnded = func(info *CallInfo) {
		eventsMu.Lock()
		events = append(events, "ended:"+string(info.StateData.EndReason))
		eventsMu.Unlock()
		close(done)
	}

	peer := types.NewJID("5511999990001", types.DefaultUserServer)
	m.HandleCallOffer(context.Background(), offerNode("CID_ORDER", peer.String()), peer)
	m.AbandonCall(core.EndCallReasonUserEnded)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout esperando hooks assíncronos")
	}

	eventsMu.Lock()
	defer eventsMu.Unlock()

	require.Len(t, events, 4)
	assert.Equal(t, "incoming:CID_ORDER", events[0])
	assert.Equal(t, "state:incoming_ringing", events[1])
	assert.Equal(t, "state:ended", events[2])
	assert.Equal(t, "ended:user_ended", events[3])
}

func TestCallManager_AsyncHooks_LockInversionFree_NoDeadlock(t *testing.T) {
	sock := &fakeSock{}
	m := NewCallManager(sock, slog.Default())
	m.DeferPreaccept = true
	defer m.Close()

	endedDone := make(chan struct{})

	// Dentro dos hooks, chamamos métodos de CallManager que travam m.mu.
	// Se os hooks fossem chamados de forma síncrona com m.mu travado, isso geraria deadlock.
	m.OnIncoming = func(info *CallInfo) {
		_ = m.CurrentCall()
	}

	m.OnStateChange = func(info *CallInfo) {
		_ = m.CurrentCall()
	}

	m.OnEnded = func(info *CallInfo) {
		_ = m.CurrentCall()
		close(endedDone)
	}

	peer := types.NewJID("5511999990001", types.DefaultUserServer)
	m.HandleCallOffer(context.Background(), offerNode("CID_DEADLOCK", peer.String()), peer)
	m.AbandonCall(core.EndCallReasonDeclined)

	select {
	case <-endedDone:
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock detectado: hooks síncronos com lock inversion")
	}
}

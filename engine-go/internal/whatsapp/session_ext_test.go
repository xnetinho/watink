package whatsapp

import (
	"errors"
	"testing"

	"go.mau.fi/whatsmeow"
)

var errSentinel = errors.New("loader unavailable")

// newMinimalService returns a WhatsAppService with only the fields needed for
// offline session-management tests (no DB, no RabbitMQ, no whatsmeow container).
func newMinimalService(loader SessionLoader) *WhatsAppService {
	svc := &WhatsAppService{
		clients:         make(map[int]*whatsmeow.Client),
		sessionLoader:   loader,
		historyRequests: make(map[string]*pendingHistory),
		groupNames:      make(map[string]string),
		groupMetaMap:    make(map[string]groupMeta),
		picCache:        make(map[string]string),
	}
	svc.publishEvent = func(_ string, _ int, _ string, _ map[string]interface{}) {}
	return svc
}

// TestStopClient_NotFound verifies that StopClient returns an error when the
// requested session ID is not in the active clients map.
func TestStopClient_NotFound(t *testing.T) {
	svc := newMinimalService(&mockSessionLoader{})
	err := svc.StopClient(99)
	if err == nil {
		t.Fatal("expected error for unknown session, got nil")
	}
}

// TestForceLogout_NonexistentSession verifies that ForceLogout returns nil even
// when the session is not in the active clients map (idempotent).
func TestForceLogout_NonexistentSession(t *testing.T) {
	svc := newMinimalService(&mockSessionLoader{})
	err := svc.ForceLogout(999)
	if err != nil {
		t.Fatalf("ForceLogout on missing session should return nil, got %v", err)
	}
}

// TestAutoRestartSessions_LoaderError verifies that AutoRestartSessions handles
// a loader error gracefully without panicking and without adding any clients.
func TestAutoRestartSessions_LoaderError(t *testing.T) {
	loader := &mockSessionLoader{err: errSentinel}
	svc := newMinimalService(loader)
	svc.AutoRestartSessions() // must not panic
	if len(svc.clients) != 0 {
		t.Errorf("no clients should be registered after a loader error, got %d", len(svc.clients))
	}
}

// TestAutoRestartSessions_EmptyLoader verifies that AutoRestartSessions is a
// no-op when there are no active sessions to restore.
func TestAutoRestartSessions_EmptyLoader(t *testing.T) {
	loader := &mockSessionLoader{sessions: []ActiveSession{}}
	svc := newMinimalService(loader)
	svc.AutoRestartSessions() // must not panic, no clients added
	if len(svc.clients) != 0 {
		t.Errorf("no clients should be registered for an empty session list, got %d", len(svc.clients))
	}
}

// O business escuta "session.pairing_code". Antes o engine publicava "session.pairingcode" e o código de pareamento
// por telefone nunca chegava ao usuário.
func TestEmitPairingCode_UsaAChaveQueOBusinessEscuta(t *testing.T) {
	svc, calls := newTestService()
	svc.emitPairingCode(7, "tenant-1", "ABCD-1234")

	if len(*calls) != 1 {
		t.Fatalf("esperava 1 evento, veio %d", len(*calls))
	}
	c := (*calls)[0]
	if c.eventType != "session.pairing_code" {
		t.Fatalf("eventType = %q, o business escuta session.pairing_code", c.eventType)
	}
	if c.payload["pairingCode"] != "ABCD-1234" || c.payload["sessionId"] != "7" || c.payload["status"] != "QRCODE" {
		t.Fatalf("payload inesperado: %v", c.payload)
	}
}

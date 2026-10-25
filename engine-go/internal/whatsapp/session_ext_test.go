package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow"
)

func newMinimalService() *WhatsAppService {
	svc := &WhatsAppService{
		clients:         make(map[int]*whatsmeow.Client),
		historyRequests: make(map[string]*pendingHistory),
		groupNames:      make(map[string]string),
		groupMetaMap:    make(map[string]groupMeta),
		picCache:        make(map[string]string),
	}
	svc.publishEvent = func(_ string, _ int, _ string, _ map[string]interface{}) {}
	return svc
}

func TestStopClient_NotFound(t *testing.T) {
	if err := newMinimalService().StopClient(99); err == nil {
		t.Fatal("expected error for unknown session")
	}
}

func TestForceLogout_NonexistentSession(t *testing.T) {
	if err := newMinimalService().ForceLogout(999); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

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

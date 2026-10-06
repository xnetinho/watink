package whatsapp

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/calls"
)

func TestHandleCallCommand_SessionWithoutCalls(t *testing.T) {
	s := &WhatsAppService{callSessions: map[int]*calls.Session{}}
	for _, cmd := range []string{"call.ready", "call.accept", "call.reject", "call.end", "call.start"} {
		if err := s.HandleCallCommand(99, cmd, CallPayload{CallID: "X"}); !errors.Is(err, ErrCallsUnavailable) {
			t.Fatalf("%s: %v", cmd, err)
		}
	}
}

func TestHandleCallCommand_UnknownCommandIsIgnored(t *testing.T) {
	s := &WhatsAppService{callSessions: map[int]*calls.Session{1: calls.NewSession(calls.SessionConfig{ID: 1})}}
	if err := s.HandleCallCommand(1, "call.nope", CallPayload{}); err != nil {
		t.Fatalf("comando desconhecido não deve falhar: %v", err)
	}
}

func TestHandleCallCommand_UnknownCallID(t *testing.T) {
	s := &WhatsAppService{callSessions: map[int]*calls.Session{1: calls.NewSession(calls.SessionConfig{ID: 1})}}
	for _, cmd := range []string{"call.ready", "call.accept", "call.reject", "call.end"} {
		if err := s.HandleCallCommand(1, cmd, CallPayload{CallID: "NAOEXISTE"}); !errors.Is(err, calls.ErrNoCall) {
			t.Fatalf("%s: %v", cmd, err)
		}
	}
}

// Contrato do payload dos comandos: o business envia {"callId","to"}.
func TestCallPayload_Contract(t *testing.T) {
	var p CallPayload
	if err := json.Unmarshal([]byte(`{"callId":"ABC","to":"5511999990001@s.whatsapp.net"}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.CallID != "ABC" || p.To != "5511999990001@s.whatsapp.net" {
		t.Fatalf("%+v", p)
	}
}

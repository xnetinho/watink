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

// Contrato business → engine: o JSON que o business publica (calls.Service.command e
// calls.Service.Place) tem que ser lido pelo engine sem perda. Os dois lados são testados em
// pacotes separados, então um erro de nome de campo passaria em ambos: este teste fixa o
// formato EXATO que o business emite (copiado de business/internal/calls/events.go e place.go).
func TestCallCommandContract_BusinessEnvelopeIsReadByEngine(t *testing.T) {
	cases := []struct {
		name, body, wantCall, wantTo string
	}{
		{
			"ready/accept/reject/end: só o callId",
			`{"id":"u","timestamp":1,"tenantId":"t","type":"call.ready","payload":{"callId":"ABC123"}}`,
			"ABC123", "",
		},
		{
			"start: callId e destino",
			`{"id":"u","timestamp":1,"tenantId":"t","type":"call.start","payload":{"callId":"DEF456","to":"5511999990001@s.whatsapp.net"}}`,
			"DEF456", "5511999990001@s.whatsapp.net",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var env struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}
			if err := json.Unmarshal([]byte(c.body), &env); err != nil {
				t.Fatal(err)
			}
			var p CallPayload
			if err := json.Unmarshal(env.Payload, &p); err != nil {
				t.Fatal(err)
			}
			if p.CallID != c.wantCall || p.To != c.wantTo {
				t.Fatalf("o engine leu %+v, esperava callId=%q to=%q", p, c.wantCall, c.wantTo)
			}
		})
	}
}

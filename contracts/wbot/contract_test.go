package wbot

import (
	"encoding/json"
	"testing"
)

func TestCommandAndEventRoutingKeys(t *testing.T) {
	if got := Command("tenant-1", "42", CommandMessageText); got != "wbot.tenant-1.42.message.send.text" {
		t.Fatalf("command routing key = %q", got)
	}
	if got := Event("tenant-1", "42", EventSessionPairingCode); got != "wbot.tenant-1.42.session.pairing_code" {
		t.Fatalf("event routing key = %q", got)
	}
}

func TestEnvelopeCarriesContractVersion(t *testing.T) {
	body, err := json.Marshal(Envelope{Version: Version, Type: EventMessageReceived, TenantID: "tenant-1"})
	if err != nil {
		t.Fatal(err)
	}
	var got Envelope
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != Version || got.Type != EventMessageReceived || got.TenantID != "tenant-1" {
		t.Fatalf("envelope = %+v", got)
	}
}

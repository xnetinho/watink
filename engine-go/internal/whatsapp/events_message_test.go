package whatsapp

import "testing"

func TestPollVotePayload_UsesBusinessContract(t *testing.T) {
	payload := pollVotePayload(7, "poll-1", "5511999999999@s.whatsapp.net", [][]byte{{0xab, 0xcd}, {0x01}})
	if payload["sessionId"] != "7" || payload["pollMessageId"] != "poll-1" || payload["voterJid"] != "5511999999999@s.whatsapp.net" {
		t.Fatalf("payload identity = %#v", payload)
	}
	if payload["optionSelected"] != "abcd,01" {
		t.Fatalf("selected options = %q", payload["optionSelected"])
	}
}

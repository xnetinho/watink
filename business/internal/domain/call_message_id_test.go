package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallMessageID_TraducaoNosDoisSentidos(t *testing.T) {
	const callID = "00FC188DC5567E7BEAB455F5826BE1CE"
	assert.Equal(t, "call:"+callID, CallMessageID(callID))
	assert.Equal(t, callID, WhatsAppMessageID(CallMessageID(callID)), "ida e volta")
	assert.Equal(t, "3EB0ABC", WhatsAppMessageID("3EB0ABC"), "mensagem comum passa igual")
	assert.Equal(t, "", WhatsAppMessageID(""))
}

func TestWatinkMessageIDs_TentaOIdDoWhatsAppEDoRegistroDeChamada(t *testing.T) {
	assert.Equal(t, []string{"ABC", "call:ABC"}, WatinkMessageIDs("ABC"))
	assert.Equal(t, []string{"call:ABC"}, WatinkMessageIDs("call:ABC"), "já é o id do registro")
}

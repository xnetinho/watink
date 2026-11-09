package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A chave da conversa de TODO canal leva o prefixo do canal ("wa:", "tg:"...). Sem prefixo, o número de um canal pode colidir com o de outro e
// ninguém sabe de que canal uma chave é só olhando para ela.
func TestConversationKey_TemPrefixoDoCanal(t *testing.T) {
	assert.Equal(t, "wa:7", WhatsAppConversationKey(7))
	assert.Equal(t, "tg:29494932", TelegramConversationKey(29494932))
	assert.Equal(t, "tg:-1001234567890", TelegramConversationKey(-1001234567890), "grupo, supergrupo e canal têm chat_id negativo")
}

func TestTelegramChatID_VoltaDaChave(t *testing.T) {
	for _, id := range []int64{29494932, -4012345678, -1001234567890} {
		got, ok := TelegramChatID(TelegramConversationKey(id))
		assert.True(t, ok)
		assert.Equal(t, id, got, "ida e volta de %d", id)
	}
}

func TestTelegramChatID_RejeitaChaveQueNaoEDoTelegram(t *testing.T) {
	for _, k := range []string{"", "wa:7", "29494932", "tg:", "tg:abc", "tg:12x", " tg:5"} {
		_, ok := TelegramChatID(k)
		assert.False(t, ok, "chave %q", k)
	}
}

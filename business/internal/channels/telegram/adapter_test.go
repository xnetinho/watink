package telegram_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/channels/telegram"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramAdapter_ContractAndParsing(t *testing.T) {
	adapter := telegram.NewAdapter(nil)

	assert.Equal(t, domain.ChannelKindTelegram, adapter.Kind())
	caps := adapter.Capabilities()
	assert.False(t, caps.InitiateConversation)
	assert.Nil(t, caps.ReplyWindow)
	assert.Equal(t, 4096, caps.MaxTextBytes)
	assert.Equal(t, domain.AuthKindToken, caps.Auth)

	validConn := &models.ChannelConnection{Status: "CONNECTED", CredentialsEnc: "fake-token"}
	disconnectedConn := &models.ChannelConnection{Status: "DISCONNECTED"}

	// Roda a suíte de contrato compartilhada (Tarefa 3.5 / ADR 0033)
	domain.RunChannelContractSuite(t, adapter, validConn, disconnectedConn)

	// Testa parse de Webhook de Inbound Message do Telegram
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/botfake-token/getMe" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":123456,"is_bot":true,"first_name":"TestBot","username":"test_bot"}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Testa ParseInbound
	updateJSON := `{
		"update_id": 10001,
		"message": {
			"message_id": 42,
			"from": {
				"id": 987654,
				"first_name": "Maria",
				"last_name": "Silva",
				"username": "mariasilva"
			},
			"chat": {
				"id": 987654,
				"type": "private"
			},
			"date": 1700000000,
			"text": "Olá, gostaria de saber mais sobre o produto."
		}
	}`

	inboundMsgs, err := adapter.ParseInbound(context.Background(), validConn, []byte(updateJSON), nil)
	require.NoError(t, err)
	require.Len(t, inboundMsgs, 1)

	msg := inboundMsgs[0]
	assert.Equal(t, domain.ChannelKindTelegram, msg.Channel)
	assert.Equal(t, "42", msg.ExternalMessageID)
	assert.Equal(t, "tg:987654", msg.ConversationKey)
	assert.Equal(t, "987654", msg.Sender.ExternalID)
	assert.Equal(t, "Maria Silva", msg.Sender.Name)
	assert.Equal(t, "Olá, gostaria de saber mais sobre o produto.", msg.Body)
	assert.Equal(t, "chat", msg.MediaType)
}

// Em grupo, supergrupo e canal o ticket é do CHAT (chat_id negativo), e quem escreve é a pessoa (from.id, positivo): são identidades diferentes.
func TestParseInbound_ChaveDaConversaEDoChatEAPessoaEDoRemetente(t *testing.T) {
	adapter := telegram.NewAdapter(nil)
	conn := &models.ChannelConnection{Kind: "telegram", Status: "CONNECTED"}
	for _, tc := range []struct {
		name, chatType, key string
		chatID              int64
	}{
		{"privado", "private", "tg:555", 555},
		{"grupo", "group", "tg:-4012345678", -4012345678},
		{"supergrupo", "supergroup", "tg:-1001234567890", -1001234567890},
	} {
		raw := fmt.Sprintf(`{"update_id":1,"message":{"message_id":9,"from":{"id":555,"first_name":"Ana"},"chat":{"id":%d,"type":%q,"title":"Time"},"date":1700000000,"text":"oi"}}`, tc.chatID, tc.chatType)
		msgs, err := adapter.ParseInbound(context.Background(), conn, []byte(raw), nil)
		require.NoError(t, err, tc.name)
		require.Len(t, msgs, 1, tc.name)
		assert.Equal(t, tc.key, msgs[0].ConversationKey, tc.name)
		assert.Equal(t, "555", msgs[0].Sender.ExternalID, "%s: quem escreve é a pessoa", tc.name)
	}
}

// Em grupo o ParseInbound entrega o título do grupo à parte: o nome do remetente é só o da pessoa (antes vinha "Grupo (Pessoa)", que virava nome de contato).
func TestParseInbound_GrupoTrazOTituloSeparadoDoNomeDaPessoa(t *testing.T) {
	conn := &models.ChannelConnection{Kind: "telegram", Status: "CONNECTED"}
	raw := `{"update_id":1,"message":{"message_id":9,"from":{"id":555,"first_name":"Ana","last_name":"Lima"},"chat":{"id":-1001,"type":"supergroup","title":"Time de Vendas"},"date":1700000000,"text":"oi"}}`
	msgs, err := telegram.NewAdapter(nil).ParseInbound(context.Background(), conn, []byte(raw), nil)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Ana Lima", msgs[0].Sender.Name)
	assert.Equal(t, "Time de Vendas", msgs[0].Sender.GroupName)
	assert.Equal(t, "-1001", msgs[0].Sender.GroupJID)
	assert.True(t, msgs[0].Sender.IsGroup)

	priv := `{"update_id":1,"message":{"message_id":9,"from":{"id":555,"first_name":"Ana"},"chat":{"id":555,"type":"private"},"date":1700000000,"text":"oi"}}`
	msgs, err = telegram.NewAdapter(nil).ParseInbound(context.Background(), conn, []byte(priv), nil)
	require.NoError(t, err)
	assert.False(t, msgs[0].Sender.IsGroup)
	assert.Empty(t, msgs[0].Sender.GroupName)
}

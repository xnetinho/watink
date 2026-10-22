package telegram_test

import (
	"context"
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
	adapter := telegram.NewAdapter()

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
	assert.Equal(t, "987654", msg.ConversationKey)
	assert.Equal(t, "987654", msg.Sender.ExternalID)
	assert.Equal(t, "Maria Silva", msg.Sender.Name)
	assert.Equal(t, "Olá, gostaria de saber mais sobre o produto.", msg.Body)
	assert.Equal(t, "chat", msg.MediaType)
}

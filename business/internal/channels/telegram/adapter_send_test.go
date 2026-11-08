package telegram

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/mediastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Regressão: o contato do Telegram não tem number; o destino tem que ser o chat da conversa.
func TestAdapterSend_UsaConversationKeyComoChatID(t *testing.T) {
	var got string
	old := http.DefaultTransport
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":7}}`)), Header: http.Header{}}, nil
	})
	defer func() { http.DefaultTransport = old }()

	conn := &models.ChannelConnection{Kind: "telegram", Status: "CONNECTED", CredentialsEnc: "fake-token"}
	res, err := NewAdapter(nil).Send(context.Background(), conn, domain.OutboundMessage{ConversationKey: "tg:29494932", Body: "oi"})
	require.NoError(t, err)
	assert.Equal(t, "7", res.ExternalMessageID)
	assert.Contains(t, got, `"chat_id":29494932`, "a Bot API recebe o chat_id puro, sem o prefixo da chave")

	_, err = NewAdapter(nil).Send(context.Background(), conn, domain.OutboundMessage{Body: "oi"})
	assert.Error(t, err)
}

// Mídia local (/public/media/...) não é alcançável pelo Telegram: tem que subir como multipart, não como URL.
func TestClientSendPhoto_MidiaLocalSobeComoMultipart(t *testing.T) {
	t.Chdir(t.TempDir())
	var ct string
	old := http.DefaultTransport
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		ct = r.Header.Get("Content-Type")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":9}}`)), Header: http.Header{}}, nil
	})
	defer func() { http.DefaultTransport = old }()

	url, err := mediastore.SaveMediaReader(strings.NewReader("png"), "image/png")
	require.NoError(t, err)
	id, err := NewClient("t").SendPhoto(context.Background(), "1", url, "x", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(9), id)
	assert.True(t, strings.HasPrefix(ct, "multipart/form-data"), ct)
}

// Grupo, supergrupo e canal têm chat_id negativo; o prefixo da chave não pode engolir o sinal.
func TestAdapterSend_GrupoMantemOSinalDoChatID(t *testing.T) {
	var got string
	old := http.DefaultTransport
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":8}}`)), Header: http.Header{}}, nil
	})
	defer func() { http.DefaultTransport = old }()

	conn := &models.ChannelConnection{Kind: "telegram", Status: "CONNECTED", CredentialsEnc: "fake-token"}
	_, err := NewAdapter(nil).Send(context.Background(), conn, domain.OutboundMessage{ConversationKey: "tg:-1001234567890", Body: "oi"})
	require.NoError(t, err)
	assert.Contains(t, got, `"chat_id":-1001234567890`)
}

// Uma chave que não é do Telegram nunca vira destino: antes, qualquer texto era mandado à Bot API como se fosse um chat_id.
func TestAdapterSend_RecusaChaveQueNaoEDoTelegram(t *testing.T) {
	called := false
	old := http.DefaultTransport
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":1}}`)), Header: http.Header{}}, nil
	})
	defer func() { http.DefaultTransport = old }()

	conn := &models.ChannelConnection{Kind: "telegram", Status: "CONNECTED", CredentialsEnc: "fake-token"}
	for _, key := range []string{"wa:7", "29494932", "tg:abc"} {
		_, err := NewAdapter(nil).Send(context.Background(), conn, domain.OutboundMessage{ConversationKey: key, Body: "oi"})
		assert.Error(t, err, "chave %q", key)
	}
	assert.False(t, called, "nada pode ir à Bot API com uma chave inválida")
}

// O encaminhamento leva a chave do chat de ORIGEM: ela também perde o prefixo antes de ir à Bot API.
func TestAdapterSend_EncaminhamentoTraduzAsDuasChaves(t *testing.T) {
	var got string
	old := http.DefaultTransport
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":3}}`)), Header: http.Header{}}, nil
	})
	defer func() { http.DefaultTransport = old }()

	conn := &models.ChannelConnection{Kind: "telegram", Status: "CONNECTED", CredentialsEnc: "fake-token"}
	_, err := NewAdapter(nil).Send(context.Background(), conn, domain.OutboundMessage{
		ConversationKey: "tg:111",
		Forward:         &domain.ForwardRef{SameConnection: true, ExternalMessageID: "55", FromConversationKey: "tg:-222"},
	})
	require.NoError(t, err)
	assert.Contains(t, got, `"chat_id":111`)
	assert.Contains(t, got, `"from_chat_id":-222`)
}

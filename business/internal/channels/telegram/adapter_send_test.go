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
	res, err := NewAdapter(nil).Send(context.Background(), conn, domain.OutboundMessage{ConversationKey: "29494932", Body: "oi"})
	require.NoError(t, err)
	assert.Equal(t, "7", res.ExternalMessageID)
	assert.Contains(t, got, `"chat_id":29494932`)

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

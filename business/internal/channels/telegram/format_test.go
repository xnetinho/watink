package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhatsAppToTelegramHTML(t *testing.T) {
	cases := map[string]string{
		"*Admin :*\nTestado": "<b>Admin :</b>\nTestado",
		"oi *a* e _b_ e ~c~": "oi <b>a</b> e <i>b</i> e <s>c</s>",
		"2 * 3 * 4":          "2 * 3 * 4",
		"snake_case_name":    "snake_case_name",
		"<script>&":          "&lt;script&gt;&amp;",
		"*<b>*":              "<b>&lt;b&gt;</b>",
		"sem formatação":     "sem formatação",
	}
	for in, want := range cases {
		assert.Equal(t, want, whatsAppToTelegramHTML(in), in)
	}
}

func TestSendMessage_EnviaComParseModeHTML(t *testing.T) {
	var got string
	old := http.DefaultTransport
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":1}}`)), Header: http.Header{}}, nil
	})
	defer func() { http.DefaultTransport = old }()

	_, err := NewClient("t").SendMessage(context.Background(), "1", "*Admin :*\nTestado", 0)
	require.NoError(t, err)
	var sent struct {
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}
	require.NoError(t, json.Unmarshal([]byte(got), &sent))
	assert.Equal(t, "HTML", sent.ParseMode)
	assert.Equal(t, "<b>Admin :</b>\nTestado", sent.Text)
}

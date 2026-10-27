package telegram

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, msgJSON string) (mediaType, body string) {
	t.Helper()
	conn := &models.ChannelConnection{Kind: "telegram", Status: "CONNECTED"}
	out, err := NewAdapter(nil).ParseInbound(context.Background(), conn, []byte(`{"update_id":1,"message":{"message_id":5,"date":1,"chat":{"id":9,"type":"private"},"from":{"id":9,"first_name":"A"},`+msgJSON+`}}`), nil)
	require.NoError(t, err)
	require.Len(t, out, 1)
	return out[0].MediaType, out[0].Body
}

func TestParseInbound_TiposQueChegavamEmBranco(t *testing.T) {
	mt, body := parse(t, `"location":{"latitude":-7.1,"longitude":-34.8}`)
	assert.Equal(t, "location", mt)
	assert.Contains(t, body, "maps.google.com/maps?q=-7.1")
	assert.Len(t, splitPipe(body), 3, "o chat divide por | em 3 partes")

	mt, body = parse(t, `"contact":{"phone_number":"+5583999","first_name":"Ana"}`)
	assert.Equal(t, "vcard", mt)
	assert.Contains(t, body, "FN:Ana")

	mt, body = parse(t, `"poll":{"question":"Almoço?","options":[{"text":"Sim"},{"text":"Não"}]}`)
	assert.Equal(t, "poll", mt)
	assert.Contains(t, body, "Almoço?")

	mt, _ = parse(t, `"video_note":{"file_id":"x","file_unique_id":"y"}`)
	assert.Equal(t, "video", mt)
}

func TestParseInbound_TipoDesconhecidoViraAviso(t *testing.T) {
	mt, body := parse(t, `"dice":{"emoji":"🎲","value":3}`)
	assert.Equal(t, "unsupported", mt)
	assert.Equal(t, "dice", body)

	mt, body = parse(t, `"sticker":{"file_id":"x","file_unique_id":"y","is_animated":true}`)
	assert.Equal(t, "unsupported", mt)
	assert.Equal(t, "sticker animado", body)
}

func TestParseInbound_TextoComumContinuaChat(t *testing.T) {
	mt, body := parse(t, `"text":"oi","entities":[{"type":"bold","offset":0,"length":2}]`)
	assert.Equal(t, "chat", mt)
	assert.Equal(t, "oi", body)
}

func splitPipe(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '|' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(out, cur)
}

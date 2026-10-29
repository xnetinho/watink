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

func TestParseInbound_StickerEstaticoEGifSaoMidia(t *testing.T) {
	mt, _ := parse(t, `"sticker":{"file_id":"x","file_unique_id":"y","is_animated":false,"is_video":false}`)
	assert.Equal(t, "sticker", mt)
	mt, _ = parse(t, `"animation":{"file_id":"x","file_unique_id":"y"},"document":{"file_id":"x","file_unique_id":"y"}`)
	assert.Equal(t, "document", mt, "GIF do Telegram vem como animation E document; vale o documento")
}

func parseUpdate(t *testing.T, kind, msgJSON string) []domainInbound {
	t.Helper()
	conn := &models.ChannelConnection{Kind: "telegram", Status: "CONNECTED"}
	raw := `{"update_id":1,"` + kind + `":{"message_id":5,"date":1,"chat":{"id":9,"type":"private"},"from":{"id":9,"first_name":"A"},` + msgJSON + `}}`
	out, err := NewAdapter(nil).ParseInbound(context.Background(), conn, []byte(raw), nil)
	require.NoError(t, err)
	res := make([]domainInbound, len(out))
	for i, o := range out {
		res[i] = domainInbound{Edited: o.Edited, Forwarded: o.Forwarded, Rich: o.Rich, Body: o.Body, Type: o.MediaType}
	}
	return res
}

type domainInbound struct {
	Edited, Forwarded bool
	Rich              map[string]interface{}
	Body, Type        string
}

// Updates reais capturados do bot: a localização ao vivo chega como edited_message a cada ~30s.
func TestParseInbound_LocalizacaoAoVivoEEdicao(t *testing.T) {
	first := parseUpdate(t, "message", `"location":{"latitude":-7.173401,"longitude":-34.837595,"live_period":900}`)
	require.Len(t, first, 1)
	assert.False(t, first[0].Edited)
	assert.Equal(t, true, first[0].Rich["live"])

	edit := parseUpdate(t, "edited_message", `"edit_date":1793116945,"location":{"latitude":-7.17341,"longitude":-34.837586,"live_period":900,"heading":90,"horizontal_accuracy":3.0}`)
	require.Len(t, edit, 1)
	assert.True(t, edit[0].Edited, "edição precisa atualizar a original, não ser descartada")
	assert.Contains(t, edit[0].Body, "-7.173410")
	assert.Equal(t, "location", edit[0].Type)
}

func TestParseInbound_EnqueteEstruturadaEEncaminhada(t *testing.T) {
	p := parseUpdate(t, "message", `"poll":{"id":"1","question":"Q?","options":[{"text":"A","voter_count":0},{"text":"B","voter_count":0}],"allows_multiple_answers":true,"type":"regular"}`)
	require.Len(t, p, 1)
	assert.Equal(t, "poll", p[0].Type)
	assert.Equal(t, 2, p[0].Rich["selectable"])

	f := parseUpdate(t, "message", `"text":"olha isso","forward_origin":{"type":"user","date":1}`)
	require.Len(t, f, 1)
	assert.True(t, f[0].Forwarded)
	assert.Equal(t, "chat", f[0].Type)
}

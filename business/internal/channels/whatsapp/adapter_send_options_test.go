package whatsapp_test

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/channels/whatsapp"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sentPayload(t *testing.T, msg domain.OutboundMessage) (domain.SendResult, map[string]interface{}) {
	t.Helper()
	pub := &mockPublisher{}
	res, err := whatsapp.NewAdapter(nil, pub).Send(context.Background(), &models.ChannelConnection{ID: 3, TenantID: uuid.New()}, msg)
	require.NoError(t, err)
	return res, pub.lastPayload["payload"].(map[string]interface{})
}

// Cada opção do envio que o painel oferece precisa chegar ao comando do engine: é o que impede o adaptador de mandar
// "mídia comum" quando o operador pediu visualização única, voz, figurinha ou citação com prévia.
func TestWhatsAppAdapter_Send_OpcoesDeMidiaChegamAoEngine(t *testing.T) {
	media := domain.OutboundMessage{ExternalID: "5511999990000@s.whatsapp.net", MediaType: "image", MediaURL: "/media/x", StorageID: "3EB0AAAA",
		Metadata: map[string]string{"mimeType": "image/png"}, FileName: "foto original.png", ViewOnce: true}
	res, p := sentPayload(t, media)
	assert.Equal(t, "3EB0AAAA", res.ExternalMessageID, "o id de gravação é o id publicado")
	assert.Equal(t, "3EB0AAAA", p["messageId"])
	assert.Equal(t, true, p["viewOnce"])
	assert.Equal(t, "foto original.png", p["fileName"])
	assert.Equal(t, "image/png", p["mimeType"])

	_, p = sentPayload(t, domain.OutboundMessage{ExternalID: "5511999990000@s.whatsapp.net", MediaType: "audio", MediaURL: "/media/a", Voice: true})
	assert.Equal(t, "voice", p["mediaType"], "gravação (PTT) vira mediaType=voice")
	assert.NotContains(t, p, "viewOnce")

	_, p = sentPayload(t, domain.OutboundMessage{ExternalID: "5511999990000@s.whatsapp.net", MediaType: "sticker", MediaURL: "/media/s", Sticker: true})
	assert.Equal(t, "sticker", p["mediaType"])
}

func TestWhatsAppAdapter_Send_CitacaoComPreviaEEncaminhamento(t *testing.T) {
	_, p := sentPayload(t, domain.OutboundMessage{ExternalID: "120363000000000001@g.us", Body: "resp", QuotedMessageID: "3EB0Q",
		Quoted: &domain.QuotedRef{Participant: "5511888880000@s.whatsapp.net", Body: "original", Media: map[string]interface{}{"type": "image"}}})
	assert.Equal(t, "3EB0Q", p["quotedMsgId"])
	assert.Equal(t, "5511888880000@s.whatsapp.net", p["quotedJid"], "em grupo a citada precisa de quem escreveu")
	assert.Equal(t, "original", p["quotedBody"])
	assert.Equal(t, map[string]interface{}{"type": "image"}, p["quotedMedia"])

	_, p = sentPayload(t, domain.OutboundMessage{ExternalID: "5511999990000@s.whatsapp.net", Body: "x",
		Forward: &domain.ForwardRef{Score: 4}})
	assert.Equal(t, true, p["forwarded"])
	assert.Equal(t, 5, p["forwardScore"], "contador soma um à original")

	_, p = sentPayload(t, domain.OutboundMessage{ExternalID: "5511999990000@s.whatsapp.net", Body: "x", Forward: &domain.ForwardRef{AsNew: true}})
	assert.NotContains(t, p, "forwarded", "copiar como nova não leva o selo")
}

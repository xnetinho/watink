package whatsapp_test

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/channels/whatsapp"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSessionManager struct {
	startedWA   models.Whatsapp
	usePairing  bool
	phone       string
	stoppedWA   models.Whatsapp
	startCalled bool
	stopCalled  bool
}

func (m *mockSessionManager) StartWhatsAppSession(wa models.Whatsapp, usePairing bool, phone string, force bool) error {
	m.startedWA = wa
	m.usePairing = usePairing
	m.phone = phone
	m.startCalled = true
	return nil
}

func (m *mockSessionManager) StopWhatsAppSession(wa models.Whatsapp) error {
	m.stoppedWA = wa
	m.stopCalled = true
	return nil
}

type mockPublisher struct {
	lastRoutingKey string
	lastPayload    map[string]interface{}
}

func (p *mockPublisher) PublishCommand(routingKey string, payload interface{}) error {
	p.lastRoutingKey = routingKey
	if m, ok := payload.(map[string]interface{}); ok {
		p.lastPayload = m
	}
	return nil
}

func TestWhatsAppAdapter_Capabilities(t *testing.T) {
	adapter := whatsapp.NewAdapter(nil)
	caps := adapter.Capabilities()
	assert.True(t, caps.InitiateConversation)
	assert.True(t, caps.HasPhone)
	assert.True(t, caps.Reactions)
	assert.Equal(t, domain.ChannelKindWhatsApp, adapter.Kind())
}

func TestWhatsAppAdapter_ConnectAndDisconnect(t *testing.T) {
	sm := &mockSessionManager{}
	adapter := whatsapp.NewAdapter(nil, sm)

	tenantID := uuid.New()
	conn := &models.ChannelConnection{
		ID:       42,
		TenantID: tenantID,
		Name:     "Zap 1",
		Status:   "DISCONNECTED",
	}

	// 1. Connect
	res, err := adapter.Connect(context.Background(), conn, domain.ConnectInput{
		Params: map[string]string{
			"usePairingCode": "true",
			"phoneNumber":    "5511999998888",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "OPENING", res.Status)
	assert.True(t, sm.startCalled)
	assert.Equal(t, 42, sm.startedWA.ID)
	assert.Equal(t, "5511999998888", sm.phone)
	assert.True(t, sm.usePairing)

	// 2. Disconnect
	err = adapter.Disconnect(context.Background(), conn)
	require.NoError(t, err)
	assert.True(t, sm.stopCalled)
	assert.Equal(t, 42, sm.stoppedWA.ID)
}

func TestWhatsAppAdapter_SendTextWithCitationAndMentions(t *testing.T) {
	pub := &mockPublisher{}
	adapter := whatsapp.NewAdapter(nil, pub)

	tenantID := uuid.New()
	conn := &models.ChannelConnection{
		ID:       10,
		TenantID: tenantID,
	}

	msg := domain.OutboundMessage{
		ExternalID:      "5511999990000@s.whatsapp.net",
		Body:            "Resposta citando",
		QuotedMessageID: "3EB0QUOTED123",
		Metadata: map[string]string{
			"quotedJid":  "5511888880000@s.whatsapp.net",
			"quotedBody": "Mensagem original",
			"mentions":   "5511777770000@s.whatsapp.net, 5511666660000@s.whatsapp.net",
		},
	}

	res, err := adapter.Send(context.Background(), conn, msg)
	require.NoError(t, err)
	assert.Equal(t, "sent", res.Status)
	assert.NotEmpty(t, res.ExternalMessageID)

	expectedKey := wbot.Command(tenantID.String(), "10", wbot.CommandMessageText)
	assert.Equal(t, expectedKey, pub.lastRoutingKey)

	payload := pub.lastPayload["payload"].(map[string]interface{})
	assert.Equal(t, "Resposta citando", payload["body"])
	assert.Equal(t, "3EB0QUOTED123", payload["quotedMsgId"])
	assert.Equal(t, "5511888880000@s.whatsapp.net", payload["quotedJid"])
	assert.Equal(t, "Mensagem original", payload["quotedBody"])
	assert.Equal(t, []string{"5511777770000@s.whatsapp.net", "5511666660000@s.whatsapp.net"}, payload["mentions"])
}

func TestWhatsAppAdapter_SendMediaWithMimeAndData(t *testing.T) {
	pub := &mockPublisher{}
	adapter := whatsapp.NewAdapter(nil, pub)

	tenantID := uuid.New()
	conn := &models.ChannelConnection{
		ID:       12,
		TenantID: tenantID,
	}

	msg := domain.OutboundMessage{
		ExternalID: "5511999990000@s.whatsapp.net",
		Body:       "Legenda da imagem",
		MediaType:  "image",
		MediaURL:   "https://example.com/foto.jpg",
		MediaData:  []byte("fake-image-bytes"),
		Metadata: map[string]string{
			"mimeType": "image/jpeg",
			"fileName": "foto.jpg",
		},
	}

	res, err := adapter.Send(context.Background(), conn, msg)
	require.NoError(t, err)
	assert.Equal(t, "sent", res.Status)

	expectedKey := wbot.Command(tenantID.String(), "12", wbot.CommandMessageMedia)
	assert.Equal(t, expectedKey, pub.lastRoutingKey)

	payload := pub.lastPayload["payload"].(map[string]interface{})
	assert.Equal(t, "image", payload["mediaType"])
	assert.Equal(t, "image/jpeg", payload["mimeType"])
	assert.Equal(t, "foto.jpg", payload["fileName"])
	assert.NotEmpty(t, payload["mediaData"])
}

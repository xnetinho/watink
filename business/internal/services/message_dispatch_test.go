package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDispatchAdapter struct {
	caps domain.Capabilities
}

func (m *mockDispatchAdapter) Kind() domain.ChannelKind { return "mock" }
func (m *mockDispatchAdapter) Capabilities() domain.Capabilities { return m.caps }
func (m *mockDispatchAdapter) Connect(ctx context.Context, conn *models.ChannelConnection, input domain.ConnectInput) (domain.ConnectResult, error) {
	return domain.ConnectResult{}, nil
}
func (m *mockDispatchAdapter) Disconnect(ctx context.Context, conn *models.ChannelConnection) error {
	return nil
}
func (m *mockDispatchAdapter) Send(ctx context.Context, conn *models.ChannelConnection, msg domain.OutboundMessage) (domain.SendResult, error) {
	return domain.SendResult{Status: "sent", ExternalMessageID: "ext_123", Timestamp: time.Now()}, nil
}
func (m *mockDispatchAdapter) ResolveAddress(ctx context.Context, conn *models.ChannelConnection, contact *models.Contact) (string, error) {
	return "mock_addr", nil
}
func (m *mockDispatchAdapter) ParseInbound(ctx context.Context, conn *models.ChannelConnection, raw []byte, headers map[string]string) ([]domain.InboundMessage, error) {
	return nil, nil
}

func TestMessageDispatchService_CapabilitiesAndPolicy(t *testing.T) {
	reg := domain.NewChannelRegistry()
	adapter := &mockDispatchAdapter{
		caps: domain.Capabilities{
			InitiateConversation: true,
			MaxTextBytes: 50,
			Media:        []domain.MediaKind{domain.MediaKindImage},
		},
	}
	reg.Register(adapter)
	svc := services.NewMessageDispatchService(reg)

	conn := &models.ChannelConnection{
		Kind:   "mock",
		Status: "CONNECTED",
	}

	// 1. Excesso de texto deve ser recusado
	longText := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent."
	_, err := svc.Dispatch(context.Background(), conn, domain.OutboundMessage{Body: longText}, nil, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "excede o limite maximo")

	// 2. Mídia não suportada deve ser recusada
	_, err = svc.Dispatch(context.Background(), conn, domain.OutboundMessage{Body: "ok", MediaType: "video"}, nil, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nao suportado pelo canal")

	// 3. Conexão desconectada viola ReplyPolicy
	connDisc := &models.ChannelConnection{Kind: "mock", Status: "DISCONNECTED"}
	_, err = svc.Dispatch(context.Background(), connDisc, domain.OutboundMessage{Body: "ok"}, nil, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "envio bloqueado por politica do canal")

	// 4. Envio válido
	res, err := svc.Dispatch(context.Background(), conn, domain.OutboundMessage{Body: "Texto curto", MediaType: "image"}, nil, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "sent", res.Status)
	assert.Equal(t, "ext_123", res.ExternalMessageID)
}

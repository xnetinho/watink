package domain_test

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockChannel struct {
	kind domain.ChannelKind
	caps domain.Capabilities
}

func (m *mockChannel) Kind() domain.ChannelKind {
	return m.kind
}

func (m *mockChannel) Capabilities() domain.Capabilities {
	return m.caps
}

func (m *mockChannel) Connect(_ context.Context, _ *models.ChannelConnection, _ domain.ConnectInput) (domain.ConnectResult, error) {
	return domain.ConnectResult{Connected: true, Status: "CONNECTED"}, nil
}

func (m *mockChannel) Disconnect(_ context.Context, _ *models.ChannelConnection) error {
	return nil
}

func (m *mockChannel) Send(_ context.Context, _ *models.ChannelConnection, _ domain.OutboundMessage) (domain.SendResult, error) {
	return domain.SendResult{ExternalMessageID: "msg-123", Status: "sent"}, nil
}

func (m *mockChannel) ParseInbound(_ context.Context, _ *models.ChannelConnection, _ []byte, _ map[string]string) ([]domain.InboundMessage, error) {
	return nil, nil
}

func TestChannelRegistry(t *testing.T) {
	registry := domain.NewChannelRegistry()

	mockTG := &mockChannel{
		kind: domain.ChannelKindTelegram,
		caps: domain.Capabilities{
			MaxTextBytes: 4096,
			Auth:         domain.AuthKindToken,
			HasPhone:     false,
		},
	}

	registry.Register(mockTG)

	found, err := registry.Get(domain.ChannelKindTelegram)
	require.NoError(t, err)
	assert.Equal(t, domain.ChannelKindTelegram, found.Kind())
	assert.Equal(t, 4096, found.Capabilities().MaxTextBytes)

	_, err = registry.Get(domain.ChannelKindTikTok)
	require.Error(t, err)

	list := registry.List()
	assert.Len(t, list, 1)
}

package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDispatchChannel struct {
	sent []domain.OutboundMessage
}

func (m *mockDispatchChannel) Kind() domain.ChannelKind { return "telegram" }
func (m *mockDispatchChannel) Capabilities() domain.Capabilities {
	return domain.Capabilities{InitiateConversation: true, MaxTextBytes: 4096}
}
func (m *mockDispatchChannel) Connect(ctx context.Context, conn *models.ChannelConnection, input domain.ConnectInput) (domain.ConnectResult, error) {
	return domain.ConnectResult{Connected: true, Status: "CONNECTED"}, nil
}
func (m *mockDispatchChannel) Disconnect(ctx context.Context, conn *models.ChannelConnection) error {
	return nil
}
func (m *mockDispatchChannel) Send(ctx context.Context, conn *models.ChannelConnection, msg domain.OutboundMessage) (domain.SendResult, error) {
	m.sent = append(m.sent, msg)
	return domain.SendResult{ExternalMessageID: "tg-9999", Status: "sent", Timestamp: time.Now()}, nil
}
func (m *mockDispatchChannel) ResolveAddress(ctx context.Context, conn *models.ChannelConnection, contact *models.Contact) (string, error) {
	return contact.Number, nil
}
func (m *mockDispatchChannel) ParseInbound(ctx context.Context, conn *models.ChannelConnection, raw []byte, headers map[string]string) ([]domain.InboundMessage, error) {
	return nil, nil
}

func TestPluginSendTicketMessage_AgnosticChannelDispatch(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	ch := &mockDispatchChannel{}
	domain.DefaultChannelRegistry().Register(ch)

	conn := models.ChannelConnection{
		TenantID: tenantID,
		Kind:     "telegram",
		Name:     "Telegram Support",
		Status:   "CONNECTED",
	}
	require.NoError(t, db.Create(&conn).Error)

	contact := models.Contact{
		TenantID: tenantID,
		Name:     "Cliente Telegram",
		Number:   "12345678",
	}
	require.NoError(t, db.Create(&contact).Error)

	ticket := models.Ticket{
		TenantID:            tenantID,
		ContactID:           contact.ID,
		ChannelConnectionID: &conn.ID,
		ConversationKey:     "tg_chat_1",
		Status:              "open",
	}
	require.NoError(t, db.Create(&ticket).Error)

	core := &coreImpl{db: db}

	err := core.SendTicketMessage(tenantID, ticket.ID, "Mensagem do Plugin")
	require.NoError(t, err)

	require.Len(t, ch.sent, 1)
	assert.Equal(t, "Mensagem do Plugin", ch.sent[0].Body)
	assert.Equal(t, "12345678", ch.sent[0].ExternalID)

	var saved models.Message
	require.NoError(t, db.Where(`"ticketId" = ? AND "tenantId" = ?`, ticket.ID, tenantID).First(&saved).Error)
	assert.Equal(t, "tg-9999", saved.ExternalID)
	expectedID := domain.StorageMessageID(domain.ChannelKindTelegram, conn.ID, ticket.ConversationKey, "tg-9999")
	assert.Equal(t, expectedID, saved.ID)
}

package repository_test

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Toda conversa, de qualquer canal, é identificada por (conexão, chave da conversa), e o índice único de conversa aberta vale
// para todas. No WhatsApp a chave é o contato (domain.WhatsAppConversationKey); no Telegram é a do chat.
func TestChannelTicketFoundation_UmaConversaAbertaPorChave(t *testing.T) {
	db := testutil.NewTestDB(t)

	// Cria o índice parcial único explicitamente no banco de teste (em produção vem de addCustomIndexes).
	require.NoError(t, db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tickets_open_conversation
		ON "Tickets" ("tenantId", "channelConnectionId", "conversationKey")
		WHERE "status" IN ('open', 'pending')
		  AND "conversationKey" IS NOT NULL
		  AND "conversationKey" != ''
	`).Error)

	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "Tenant Teste Channel Tickets"}).Error)

	whatsapp := models.Whatsapp{Name: "WhatsApp Conexão 1", TenantID: tenantID, Status: "CONNECTED"}
	require.NoError(t, models.CreateWhatsapp(db, &whatsapp))
	telegram := models.ChannelConnection{TenantID: tenantID, Kind: "telegram", Name: "Telegram Bot 1", Status: "CONNECTED"}
	require.NoError(t, db.Create(&telegram).Error)

	contact := models.Contact{Name: "Contato Teste", Number: "5511988887777", TenantID: tenantID}
	require.NoError(t, db.Create(&contact).Error)

	repo := repository.NewGORMTicketRepo(db)
	ctx := context.Background()

	cases := []struct {
		nome string
		conn int
		key  string
	}{
		{"whatsapp", whatsapp.ID, domain.WhatsAppConversationKey(contact.ID)},
		{"telegram", telegram.ID, "chat_123456"},
	}
	for _, c := range cases {
		t.Run(c.nome, func(t *testing.T) {
			first := models.Ticket{TenantID: tenantID, ContactID: contact.ID, ChannelConnectionID: c.conn, ConversationKey: c.key, Status: "open"}
			require.NoError(t, db.Create(&first).Error)

			found, err := repo.FindOpenByConversation(ctx, tenantID, c.conn, c.key)
			require.NoError(t, err)
			require.NotNil(t, found)
			assert.Equal(t, first.ID, found.ID)

			dup := models.Ticket{TenantID: tenantID, ContactID: contact.ID, ChannelConnectionID: c.conn, ConversationKey: c.key, Status: "pending"}
			require.Error(t, db.Create(&dup).Error, "segunda conversa aberta com a mesma chave deve violar idx_tickets_open_conversation")

			// Fechado o anterior, uma nova conversa aberta na mesma chave é permitida.
			require.NoError(t, db.Model(&first).Update("status", "closed").Error)
			again := models.Ticket{TenantID: tenantID, ContactID: contact.ID, ChannelConnectionID: c.conn, ConversationKey: c.key, Status: "open"}
			require.NoError(t, db.Create(&again).Error)
		})
	}
}

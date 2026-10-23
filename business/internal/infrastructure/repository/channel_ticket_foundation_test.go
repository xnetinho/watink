package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/database"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelTicketFoundation_ParityAndIndexes(t *testing.T) {
	db := testutil.NewTestDB(t)

	// Cria o índice parcial único explicitamente no banco de teste
	require.NoError(t, db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tickets_open_conversation 
		ON "Tickets" ("tenantId", "channelConnectionId", "conversationKey") 
		WHERE "status" IN ('open', 'pending') 
		  AND "channelConnectionId" IS NOT NULL 
		  AND "conversationKey" IS NOT NULL 
		  AND "conversationKey" != ''
	`).Error)

	tenantID := uuid.New()
	tenant := models.Tenant{ID: tenantID, Name: "Tenant Teste Channel Tickets"}
	require.NoError(t, db.Create(&tenant).Error)

	whatsapp := models.Whatsapp{
		Name:     "WhatsApp Conexão 1",
		TenantID: tenantID,
		Status:   "CONNECTED",
	}
	require.NoError(t, db.Create(&whatsapp).Error)

	contact := models.Contact{
		Name:     "Contato Teste",
		Number:   "5511988887777",
		TenantID: tenantID,
	}
	require.NoError(t, db.Create(&contact).Error)

	// 1. Criar ticket legado (sem conversationKey e sem channelConnectionId)
	legacyTicket := models.Ticket{
		TenantID:   tenantID,
		ContactID:  contact.ID,
		WhatsappID: whatsapp.ID,
		Status:     "open",
	}
	require.NoError(t, db.Create(&legacyTicket).Error)

	// 2. Provar paridade de FindOpenByContact no repositório antes e depois do backfill
	repo := repository.NewGORMTicketRepo(db)
	ctx := context.Background()

	found, err := repo.FindOpenByContact(ctx, tenantID, contact.ID, whatsapp.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, legacyTicket.ID, found.ID)
	assert.Equal(t, "open", found.Status)

	// 3. Executar o Backfill de conversationKey
	require.NoError(t, database.BackfillTicketConversationKeys(db))

	var updatedTicket models.Ticket
	require.NoError(t, db.First(&updatedTicket, legacyTicket.ID).Error)
	assert.NotEmpty(t, updatedTicket.ConversationKey, "conversationKey deve ter sido preenchida pelo backfill")
	assert.Equal(t, fmt.Sprintf("%d", contact.ID), updatedTicket.ConversationKey)

	// 4. Paridade pós-backfill: FindOpenByContact continua retornando o mesmo ticket intacto
	foundAfter, err := repo.FindOpenByContact(ctx, tenantID, contact.ID, whatsapp.ID)
	require.NoError(t, err)
	require.NotNil(t, foundAfter)
	assert.Equal(t, legacyTicket.ID, foundAfter.ID)
	assert.Equal(t, updatedTicket.ConversationKey, foundAfter.ConversationKey)

	// 5. Validar regra do índice único de conversa aberta por canal (Task 2.4)
	channelConn := models.ChannelConnection{
		TenantID: tenantID,
		Kind:     "telegram",
		Name:     "Telegram Bot 1",
		Status:   "CONNECTED",
	}
	require.NoError(t, db.Create(&channelConn).Error)

	convKey := "chat_123456"
	t1 := models.Ticket{
		TenantID:            tenantID,
		ContactID:           contact.ID,
		ChannelConnectionID: &channelConn.ID,
		ConversationKey:     convKey,
		Status:              "open",
	}
	require.NoError(t, db.Create(&t1).Error)

	// Tentativa de criar outro ticket aberto com a MESMA conversationKey na mesma conexão deve violar unique
	t2Duplicate := models.Ticket{
		TenantID:            tenantID,
		ContactID:           contact.ID,
		ChannelConnectionID: &channelConn.ID,
		ConversationKey:     convKey,
		Status:              "pending",
	}
	err = db.Create(&t2Duplicate).Error
	require.Error(t, err, "deve violar idx_tickets_open_conversation")

	// Mas se t1 for fechado ('closed'), um novo ticket aberto ('pending') com a mesma chave deve ser permitido
	require.NoError(t, db.Model(&t1).Update("status", "closed").Error)

	t3NewOpen := models.Ticket{
		TenantID:            tenantID,
		ContactID:           contact.ID,
		ChannelConnectionID: &channelConn.ID,
		ConversationKey:     convKey,
		Status:              "open",
	}
	require.NoError(t, db.Create(&t3NewOpen).Error, "após fechar o ticket anterior, novo ticket aberto na mesma conversa deve ser permitido")
}

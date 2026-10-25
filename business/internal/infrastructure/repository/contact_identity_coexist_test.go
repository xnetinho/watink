package repository

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContacts_TelegramSemTelefoneCoexistem(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	// 1. Criar dois contatos sem telefone (number vazio) no mesmo tenant
	c1 := models.Contact{
		Name:     "User Telegram 1",
		Number:   "",
		TenantID: tenantID,
	}
	require.NoError(t, db.Create(&c1).Error)

	c2 := models.Contact{
		Name:     "User Telegram 2",
		Number:   "",
		TenantID: tenantID,
	}
	require.NoError(t, db.Create(&c2).Error, "contatos com number vazio devem coexistir no mesmo tenant sem violar unique")

	assert.NotEqual(t, c1.ID, c2.ID)

	// 2. Criar identidades de canal em ContactIdentities para cada um
	id1 := models.ContactIdentity{
		TenantID:    tenantID,
		ContactID:   c1.ID,
		Channel:     "telegram",
		ExternalID:  "tg_111111",
		DisplayName: "User Telegram 1",
	}
	require.NoError(t, db.Create(&id1).Error)

	id2 := models.ContactIdentity{
		TenantID:    tenantID,
		ContactID:   c2.ID,
		Channel:     "telegram",
		ExternalID:  "tg_222222",
		DisplayName: "User Telegram 2",
	}
	require.NoError(t, db.Create(&id2).Error)

	// Validação de consulta e integridade
	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Where(`"tenantId" = ? AND (number = '' OR number IS NULL)`, tenantID).Count(&count).Error)
	assert.Equal(t, int64(2), count)

	var idCount int64
	require.NoError(t, db.Model(&models.ContactIdentity{}).Where(`"tenantId" = ? AND channel = 'telegram'`, tenantID).Count(&idCount).Error)
	assert.Equal(t, int64(2), idCount)
}

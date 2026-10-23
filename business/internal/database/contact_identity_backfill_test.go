package database_test

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/database"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackfillContactIdentities_IdempotentAndComplete(t *testing.T) {
	db := testutil.NewTestDB(t)

	tenantID := uuid.New()
	tenant := models.Tenant{ID: tenantID, Name: "Tenant Teste Backfill"}
	require.NoError(t, db.Create(&tenant).Error)

	lidValue := "5511999990001@lid"

	contacts := []models.Contact{
		{
			Name:     "Contato Somente Numero",
			Number:   "5511999990001",
			TenantID: tenantID,
		},
		{
			Name:     "Contato Numero e Lid",
			Number:   "5511999990002",
			Lid:      &lidValue,
			TenantID: tenantID,
		},
		{
			Name:     "Contato Sem Identificador",
			Number:   "",
			TenantID: tenantID,
		},
	}
	require.NoError(t, db.Create(&contacts).Error)

	// Primeira execução do backfill
	err := database.BackfillContactIdentities(db)
	require.NoError(t, err)

	var identities []models.ContactIdentity
	err = db.Where(`"tenantId" = ?`, tenantID).Find(&identities).Error
	require.NoError(t, err)

	// Esperamos exatamente 3 identidades criadas:
	// - 1 para o contato 1 (number)
	// - 2 para o contato 2 (number + lid)
	// - 0 para o contato 3
	assert.Len(t, identities, 3)

	identMap := make(map[string]models.ContactIdentity)
	for _, id := range identities {
		assert.Equal(t, "whatsapp", id.Channel)
		identMap[id.ExternalID] = id
	}

	assert.Contains(t, identMap, "5511999990001")
	assert.Equal(t, contacts[0].ID, identMap["5511999990001"].ContactID)
	assert.Equal(t, "Contato Somente Numero", identMap["5511999990001"].DisplayName)

	assert.Contains(t, identMap, "5511999990002")
	assert.Equal(t, contacts[1].ID, identMap["5511999990002"].ContactID)

	assert.Contains(t, identMap, lidValue)
	assert.Equal(t, contacts[1].ID, identMap[lidValue].ContactID)

	// Segunda execução para provar IDEMPOTÊNCIA
	err = database.BackfillContactIdentities(db)
	require.NoError(t, err, "segunda execução do backfill não deve falhar")

	var identitiesAfter []models.ContactIdentity
	err = db.Where(`"tenantId" = ?`, tenantID).Find(&identitiesAfter).Error
	require.NoError(t, err)
	assert.Len(t, identitiesAfter, 3, "não deve duplicar nenhuma identidade")
}

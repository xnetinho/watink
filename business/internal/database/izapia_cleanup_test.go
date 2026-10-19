package database

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDropIzapiaSchema_NormalizesLegacyProviderAndDropsItsSchemaIdempotently(t *testing.T) {
	db := testutil.NewTestDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE "IzapiaConfigs" (id integer PRIMARY KEY)`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE "Whatsapps" ADD COLUMN "izapiaSessionId" text`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE "Whatsapps" ADD COLUMN "izapiaWebhookSecretEnc" text`).Error)
	legacy := models.Whatsapp{Name: "legacy-izapia", EngineType: "izapia", TenantID: uuid.New()}
	require.NoError(t, db.Create(&legacy).Error)

	require.NoError(t, dropIzapiaSchema(db))
	require.NoError(t, dropIzapiaSchema(db), "cleanup is safe to repeat on later application starts")

	var engineType string
	require.NoError(t, db.Raw(`SELECT "engineType" FROM "Whatsapps" WHERE id = ?`, legacy.ID).Scan(&engineType).Error)
	require.Equal(t, "whatsmeow", engineType)

	var tableCount int
	require.NoError(t, db.Raw(`SELECT count(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = 'IzapiaConfigs'`).Scan(&tableCount).Error)
	require.Zero(t, tableCount)

	var columnCount int
	require.NoError(t, db.Raw(`SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'Whatsapps' AND column_name IN ('izapiaSessionId', 'izapiaWebhookSecretEnc')`).Scan(&columnCount).Error)
	require.Zero(t, columnCount)
}

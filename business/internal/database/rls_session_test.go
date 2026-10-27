package database

import (
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type rlsFakeRedis struct{ domain.RedisService }

func (rlsFakeRedis) SetLock(string, string, time.Duration) (bool, error) { return true, nil }
func (rlsFakeRedis) DelLock(string) error                                { return nil }

type rlsFakePublisher struct{ keys []string }

func (p *rlsFakePublisher) PublishCommand(key string, _ interface{}) error {
	p.keys = append(p.keys, key)
	return nil
}

// Regressão do "Internal server error" ao clicar em Conectar: o serviço de sessão usava o handle global sem escopo de
// empresa, e o roteador RLS recusava o UPDATE. Roda contra o banco migrado de verdade (papel sem BYPASSRLS).
func TestStartWhatsAppSession_ComRLSReal(t *testing.T) {
	admin, app, _, _ := migratedDB(t)
	tenant := uuid.New()
	_, err := admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'A','active',now(),now())`, tenant)
	require.NoError(t, err)
	require.NoError(t, seedRow(admin, "Whatsapps", tenant))
	var id int
	require.NoError(t, admin.QueryRow(`SELECT id FROM "Whatsapps" WHERE "tenantId" = $1`, tenant).Scan(&id))

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: routerOver(t, app)}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)

	pub := &rlsFakePublisher{}
	wss := services.NewWhatsAppSessionService(gdb, pub, rlsFakeRedis{}, domain.BroadcastOrNop(nil))
	wa := models.Whatsapp{ID: id, TenantID: tenant, Status: "DISCONNECTED", EngineType: "whatsmeow"}

	require.NoError(t, wss.StartWhatsAppSession(wa, false, "", true))
	assert.Len(t, pub.keys, 1, "o comando session.start precisa chegar ao broker")

	var status string
	require.NoError(t, admin.QueryRow(`SELECT status FROM "Whatsapps" WHERE id = $1`, id).Scan(&status))
	assert.Equal(t, "OPENING", status)

}

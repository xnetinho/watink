package database

import (
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCallLog(tenant uuid.UUID, callID string) models.CallLog {
	return models.CallLog{TenantID: tenant, CallID: callID, WhatsappID: 1, Direction: "incoming", Status: "ringing", StartedAt: time.Now()}
}

// 5.1: o callId é único POR EMPRESA. A mesma duplicata na mesma empresa é
// rejeitada pelo banco; em outra empresa é aceita.
func TestCallLog_CallIDIsUniquePerTenant(t *testing.T) {
	db := testutil.NewTestDB(t)
	a, b := uuid.New(), uuid.New()

	first := newCallLog(a, "CALL-1")
	require.NoError(t, db.Create(&first).Error)

	dup := newCallLog(a, "CALL-1")
	assert.Error(t, db.Create(&dup).Error, "mesma empresa + mesmo callId deve ser rejeitado pelo banco")

	other := newCallLog(b, "CALL-1")
	assert.NoError(t, db.Create(&other).Error, "outra empresa pode ter o mesmo callId")

	next := newCallLog(a, "CALL-2")
	assert.NoError(t, db.Create(&next).Error)
}

func TestCallLog_IndexesExist(t *testing.T) {
	db := testutil.NewTestDB(t)
	var names []string
	require.NoError(t, db.Raw(`SELECT indexname FROM pg_indexes WHERE schemaname = current_schema() AND tablename = 'CallLogs'`).Scan(&names).Error)
	assert.Contains(t, names, "idx_calllogs_tenant_call")
	assert.Contains(t, names, "idx_calllogs_tenant_started")
}

func TestCallLog_DefaultsAndOptionalFields(t *testing.T) {
	db := testutil.NewTestDB(t)
	l := models.CallLog{TenantID: uuid.New(), CallID: "X", WhatsappID: 3, Direction: "outgoing", StartedAt: time.Now()}
	require.NoError(t, db.Create(&l).Error)
	var got models.CallLog
	require.NoError(t, db.First(&got, l.ID).Error)
	assert.Equal(t, "ringing", got.Status)
	assert.Zero(t, got.DurationSec)
	assert.Nil(t, got.ContactID)
	assert.Nil(t, got.TicketID)
	assert.Nil(t, got.HandledByUserID)
	assert.Nil(t, got.MosEstimated)
}

// 5.2: a auditoria de gravação é isolada por empresa.
func TestCallRecordingAccess_IsTenantScoped(t *testing.T) {
	db := testutil.NewTestDB(t)
	a, b := uuid.New(), uuid.New()
	ua := models.User{Name: "A", Email: "a@a.io", TenantID: a}
	ub := models.User{Name: "B", Email: "b@b.io", TenantID: b}
	require.NoError(t, db.Create(&ua).Error)
	require.NoError(t, db.Create(&ub).Error)

	require.NoError(t, db.Create(&models.CallRecordingAccess{TenantID: a, CallID: "C1", UserID: ua.ID, Action: "listen", At: time.Now()}).Error)
	require.NoError(t, db.Create(&models.CallRecordingAccess{TenantID: b, CallID: "C1", UserID: ub.ID, Action: "listen", At: time.Now()}).Error)
	require.NoError(t, db.Create(&models.CallRecordingAccess{TenantID: a, CallID: "C1", UserID: ua.ID, Action: "delete", At: time.Now()}).Error)

	var forA []models.CallRecordingAccess
	require.NoError(t, db.Where(`"tenantId" = ? AND "callId" = ?`, a, "C1").Find(&forA).Error)
	assert.Len(t, forA, 2)
	for _, r := range forA {
		assert.Equal(t, a, r.TenantID, "nenhum registro de outra empresa pode vazar")
	}
}

// 5.3: as cinco permissões entram no catálogo com descrição em pt, e rodar o
// Seed duas vezes não duplica.
func TestCallPermissions_AreInTheCatalogWithPortugueseDescriptions(t *testing.T) {
	db := testutil.NewTestDB(t)
	prev := DB
	DB = db
	t.Cleanup(func() { DB = prev })

	Seed()
	Seed()

	var perms []models.Permission
	require.NoError(t, db.Where("resource = ?", "calls").Find(&perms).Error)
	got := map[string]string{}
	for _, p := range perms {
		got[p.Action] = p.Description
	}
	require.Len(t, perms, 5, "exatamente 5 permissões, sem duplicar no segundo Seed")
	for _, action := range []string{"receive", "place", "read", "delete", "manage"} {
		assert.NotEmpty(t, got[action], "calls:%s sem descrição", action)
	}
	assert.Contains(t, got["receive"], "chamadas")
}

// 5.3: o Seed NÃO anexa as permissões novas a nenhum cargo já existente — só o
// alcance de empresa passa sozinho.
func TestCallPermissions_SeedDoesNotAttachToExistingCargos(t *testing.T) {
	db := testutil.NewTestDB(t)
	prev := DB
	DB = db
	t.Cleanup(func() { DB = prev })

	tenant := uuid.New()
	var cargos []models.Cargo
	for _, name := range []string{"Atendente", "Gestor", "Gerente Geral", "Administrador", "Customizado"} {
		c := models.Cargo{Name: name, TenantID: tenant}
		require.NoError(t, db.Create(&c).Error)
		cargos = append(cargos, c)
	}
	Seed()

	var n int64
	require.NoError(t, db.Table("cargo_permissoes").
		Joins(`JOIN "Permissions" p ON p.id = cargo_permissoes."permissionId"`).
		Where("p.resource = ?", "calls").Count(&n).Error)
	assert.Zero(t, n, "nenhum cargo existente pode ganhar calls:* sozinho na atualização")
}

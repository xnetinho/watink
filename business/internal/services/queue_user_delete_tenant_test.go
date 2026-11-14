package services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
)

// Sem RLS (handle simples), o filtro explícito de empresa é a única barreira: é ele que impede mover tickets para fila de OUTRA empresa e dar a carteira a
// usuário de outra empresa. Com RLS real a política do banco já esconde a linha; os dois níveis valem (defesa em profundidade, ADR 0001).
func TestDeleteQueue_FiltroDeEmpresaNaoDependeDoRLS(t *testing.T) {
	db := testutil.NewTestDB(t)
	a, b := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: a, Name: "A"}).Error)
	require.NoError(t, db.Create(&models.Tenant{ID: b, Name: "B"}).Error)
	mine := models.Queue{Name: "minha", Color: "#000", TenantID: a}
	foreign := models.Queue{Name: "alheia", Color: "#000", TenantID: b}
	require.NoError(t, db.Create(&mine).Error)
	require.NoError(t, db.Create(&foreign).Error)
	require.NoError(t, db.Create(&models.Ticket{Status: "pending", TenantID: a, QueueID: &mine.ID, ChannelConnectionID: 1}).Error)

	err := services.DeleteQueue(context.Background(), db, a, mine.ID, &foreign.ID)
	assert.ErrorIs(t, err, services.ErrQueueMoveInvalid, "fila de outra empresa não é destino")
	assert.ErrorIs(t, services.DeleteQueue(context.Background(), db, a, foreign.ID, nil), services.ErrQueueNotFound, "fila de outra empresa não se exclui")
	var still int64
	db.Model(&models.Queue{}).Where("id = ?", foreign.ID).Count(&still)
	assert.EqualValues(t, 1, still, "a fila alheia continua")
	var n int64
	db.Model(&models.Ticket{}).Where(`"queueId" = ?`, mine.ID).Count(&n)
	assert.EqualValues(t, 1, n, "o ticket não foi para a fila alheia")
}

func TestDeleteUser_FiltroDeEmpresaNaoDependeDoRLS(t *testing.T) {
	db := testutil.NewTestDB(t)
	a, b := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: a, Name: "A"}).Error)
	require.NoError(t, db.Create(&models.Tenant{ID: b, Name: "B"}).Error)
	mine := models.User{Name: "m", Email: "m@x", PasswordHash: "h", TenantID: a}
	foreign := models.User{Name: "f", Email: "f@x", PasswordHash: "h", TenantID: b}
	require.NoError(t, db.Create(&mine).Error)
	require.NoError(t, db.Create(&foreign).Error)
	require.NoError(t, db.Create(&models.Contact{Name: "c", Number: "1", TenantID: a, WalletUserID: &mine.ID}).Error)

	assert.ErrorIs(t, services.DeleteUser(context.Background(), db, a, mine.ID, &foreign.ID), services.ErrWalletTargetInvalid, "usuário de outra empresa não recebe a carteira")
	assert.ErrorIs(t, services.DeleteUser(context.Background(), db, a, foreign.ID, nil), services.ErrUserNotFound, "usuário de outra empresa não se exclui")
	var n int64
	db.Model(&models.User{}).Where("id = ?", foreign.ID).Count(&n)
	assert.EqualValues(t, 1, n)
}

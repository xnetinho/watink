package controllers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Fila padrão da empresa (PR 2a): é ela que recebe o ticket quando a conexão
// não tem fila. Setar uma fila de outra empresa é recusado.

func TestQueueController_SetDefaultQueue_Persists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupQueueTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO "Tenants" (id, name, status) VALUES (?,?,?)`, tenantID, "T", "active").Error)
	queueID := insertQueue(t, db, tenantID, "Atendimento")

	body, _ := json.Marshal(map[string]interface{}{"queueId": queueID})
	ctrl := NewQueueController()
	c, w := setupQueueContext(t, db, tenantID, "POST", "/queue/default", body)
	ctrl.SetDefaultQueue(c)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var got *int
	require.NoError(t, db.Raw(`SELECT "defaultQueueId" FROM "Tenants" WHERE id = ?`, tenantID).Scan(&got).Error)
	require.NotNil(t, got)
	assert.Equal(t, queueID, *got)
}

func TestQueueController_SetDefaultQueue_RejectsOtherTenantQueue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupQueueTestDB(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO "Tenants" (id, name, status) VALUES (?,?,?)`, tenantA, "A", "active").Error)
	foreign := insertQueue(t, db, tenantB, "Alheia")

	body, _ := json.Marshal(map[string]interface{}{"queueId": foreign})
	ctrl := NewQueueController()
	c, w := setupQueueContext(t, db, tenantA, "POST", "/queue/default", body)
	ctrl.SetDefaultQueue(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "ERR_DEFAULT_QUEUE_INVALID")
	var got *int
	require.NoError(t, db.Raw(`SELECT "defaultQueueId" FROM "Tenants" WHERE id = ?`, tenantA).Scan(&got).Error)
	assert.Nil(t, got, "fila de outra empresa não pode virar padrão")
}

func TestQueueController_SetDefaultQueue_NullClears(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupQueueTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO "Tenants" (id, name, status) VALUES (?,?,?)`, tenantID, "T", "active").Error)
	queueID := insertQueue(t, db, tenantID, "Atendimento")
	require.NoError(t, db.Exec(`UPDATE "Tenants" SET "defaultQueueId" = ? WHERE id = ?`, queueID, tenantID).Error)

	body, _ := json.Marshal(map[string]interface{}{"queueId": nil})
	ctrl := NewQueueController()
	c, w := setupQueueContext(t, db, tenantID, "POST", "/queue/default", body)
	ctrl.SetDefaultQueue(c)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var got *int
	require.NoError(t, db.Raw(`SELECT "defaultQueueId" FROM "Tenants" WHERE id = ?`, tenantID).Scan(&got).Error)
	assert.Nil(t, got)
}

func TestQueueController_ListQueues_FlagsDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupQueueTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO "Tenants" (id, name, status) VALUES (?,?,?)`, tenantID, "T", "active").Error)
	defaultID := insertQueue(t, db, tenantID, "Padrão")
	insertQueue(t, db, tenantID, "Comercial")
	require.NoError(t, db.Exec(`UPDATE "Tenants" SET "defaultQueueId" = ? WHERE id = ?`, defaultID, tenantID).Error)

	ctrl := NewQueueController()
	c, w := setupQueueContext(t, db, tenantID, "GET", "/queue", nil)
	ctrl.ListQueues(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var queues []domain.Queue
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &queues))
	require.Len(t, queues, 2)
	for _, q := range queues {
		assert.Equal(t, q.ID == defaultID, q.IsDefault, "isDefault da fila %s", q.Name)
	}
}

func insertQueue(t *testing.T, db *gorm.DB, tenantID uuid.UUID, name string) int {
	t.Helper()
	res := db.Exec(`INSERT INTO "Queues" (name, color, "tenantId") VALUES (?,?,?)`, name, "#000000", tenantID)
	require.NoError(t, res.Error)
	var id int
	require.NoError(t, db.Raw(`SELECT LASTVAL()`).Scan(&id).Error)
	return id
}

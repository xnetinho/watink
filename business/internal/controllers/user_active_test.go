package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Desativar usuário (PR 1b): bloqueia login/refresh e tira da distribuição,
// mas nunca pode trancar a organização fora do sistema.

func insertActiveUser(t *testing.T, db *gorm.DB, tenantID uuid.UUID, email string, cargoID *int) int {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO "Users" (name, email, "passwordHash", "tenantId", "cargoId") VALUES (?,?,?,?,?)`,
		email, email, "hash", tenantID, cargoID,
	).Error)
	var id int
	require.NoError(t, db.Raw(`SELECT LASTVAL()`).Scan(&id).Error)
	return id
}

func updateActive(t *testing.T, db *gorm.DB, tenantID uuid.UUID, targetID, actorID int, active bool) (*httptest.ResponseRecorder, *mockUserRepo) {
	t.Helper()
	repo := &mockUserRepo{db: db}
	ctrl := NewUserController(repo, &mockPlanLimit{})
	payload, _ := json.Marshal(map[string]interface{}{"active": active})
	c, w := setupUserContext(t, db, tenantID, "PUT", fmt.Sprintf("/users/%d", targetID), payload)
	c.Set("userId", float64(actorID))
	c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", targetID)}}
	ctrl.UpdateUser(c)
	return w, repo
}

func userActiveInDB(t *testing.T, db *gorm.DB, id int) bool {
	t.Helper()
	var active bool
	require.NoError(t, db.Raw(`SELECT active FROM "Users" WHERE id = ?`, id).Scan(&active).Error)
	return active
}

func TestUpdateUser_DeactivateSelf_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	uid := insertActiveUser(t, db, tenantID, "self@test.com", nil)

	w, _ := updateActive(t, db, tenantID, uid, uid, false)

	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "ERR_CANNOT_DEACTIVATE_SELF")
	assert.True(t, userActiveInDB(t, db, uid), "usuário não pode ser desativado por si mesmo")
}

func TestUpdateUser_DeactivateTenantOwner_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	ownerID := insertActiveUser(t, db, tenantID, "owner@test.com", nil)
	actorID := insertActiveUser(t, db, tenantID, "admin@test.com", nil)
	require.NoError(t, db.Exec(
		`INSERT INTO "Tenants" (id, name, status, "ownerId") VALUES (?,?,?,?)`,
		tenantID, "T", "active", ownerID,
	).Error)

	w, _ := updateActive(t, db, tenantID, ownerID, actorID, false)

	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "ERR_CANNOT_DEACTIVATE_ADMIN")
	assert.True(t, userActiveInDB(t, db, ownerID), "dono do tenant continua ativo")
}

func TestUpdateUser_DeactivateLastAdmin_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	adminCargo := models.Cargo{Name: "Administrador", TenantID: tenantID}
	require.NoError(t, db.Create(&adminCargo).Error)
	adminID := insertActiveUser(t, db, tenantID, "admin@test.com", &adminCargo.ID)
	actorID := insertActiveUser(t, db, tenantID, "agent@test.com", nil)

	w, _ := updateActive(t, db, tenantID, adminID, actorID, false)

	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "ERR_CANNOT_DEACTIVATE_ADMIN")
	assert.True(t, userActiveInDB(t, db, adminID), "último Administrador continua ativo")
}

func TestUpdateUser_DeactivateAdminWithAnotherActiveAdmin_Allowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	adminCargo := models.Cargo{Name: "Administrador", TenantID: tenantID}
	require.NoError(t, db.Create(&adminCargo).Error)
	otherAdmin := insertActiveUser(t, db, tenantID, "other-admin@test.com", &adminCargo.ID)
	adminID := insertActiveUser(t, db, tenantID, "admin@test.com", &adminCargo.ID)

	w, _ := updateActive(t, db, tenantID, adminID, otherAdmin, false)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.False(t, userActiveInDB(t, db, adminID))
}

// Admin já desativado não conta como acesso: desativar o único admin ATIVO
// deixaria a organização sem ninguém capaz de entrar.
func TestUpdateUser_DeactivateOnlyActiveAdmin_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	adminCargo := models.Cargo{Name: "Administrador", TenantID: tenantID}
	require.NoError(t, db.Create(&adminCargo).Error)
	activeAdmin := insertActiveUser(t, db, tenantID, "active-admin@test.com", &adminCargo.ID)
	inactiveAdmin := insertActiveUser(t, db, tenantID, "inactive-admin@test.com", &adminCargo.ID)
	require.NoError(t, db.Exec(`UPDATE "Users" SET active = false WHERE id = ?`, inactiveAdmin).Error)
	actorID := insertActiveUser(t, db, tenantID, "agent@test.com", nil)

	w, _ := updateActive(t, db, tenantID, activeAdmin, actorID, false)

	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "ERR_CANNOT_DEACTIVATE_ADMIN")
	assert.True(t, userActiveInDB(t, db, activeAdmin))
}

func TestUpdateUser_Deactivate_BumpsTokenVersion(t *testing.T) {	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	agentID := insertActiveUser(t, db, tenantID, "agent@test.com", nil)
	actorID := insertActiveUser(t, db, tenantID, "admin@test.com", nil)

	w, repo := updateActive(t, db, tenantID, agentID, actorID, false)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.False(t, userActiveInDB(t, db, agentID), "usuário desativado no banco")
	// O refresh token antigo para de valer: bump de tokenVersion no mesmo UPDATE.
	assert.Equal(t, false, repo.lastUpdate["active"])
	assert.NotNil(t, repo.lastUpdate["tokenVersion"], "tokenVersion deve ser incrementado ao desativar")
}

func TestUpdateUser_Reactivate_KeepsTokenVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	agentID := insertActiveUser(t, db, tenantID, "agent@test.com", nil)
	actorID := insertActiveUser(t, db, tenantID, "admin@test.com", nil)
	require.NoError(t, db.Exec(`UPDATE "Users" SET active = false WHERE id = ?`, agentID).Error)

	w, repo := updateActive(t, db, tenantID, agentID, actorID, true)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.True(t, userActiveInDB(t, db, agentID))
	assert.Nil(t, repo.lastUpdate["tokenVersion"], "reativar não invalida sessão")
}

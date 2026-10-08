package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func callsRouter(db *gorm.DB, tenant uuid.UUID, userID int, alcance string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenant)
		c.Set("alcance", alcance)
		c.Set("userId", float64(userID))
		c.Next()
	})
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	for _, action := range []string{"receive", "place", "read", "delete", "manage"} {
		r.GET("/calls-"+action, auth.RequirePermission("calls", action), ok)
	}
	return r
}

func hit(r *gin.Engine, path string) int {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code
}

// 5.3: as cinco permissões de chamada são exigidas de verdade.
func TestCallsPermissions_RequirePermissionEnforcesEachAction(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	cargoID := seedCargo(t, db, tenant, "Atendente")
	receive := seedPermission(t, db, "calls", "receive")
	for _, a := range []string{"place", "read", "delete", "manage"} {
		seedPermission(t, db, "calls", a)
	}
	require.NoError(t, db.Create(&models.CargoPermissao{CargoID: cargoID, PermissionID: receive}).Error)
	user := models.User{Name: "Ana", Email: "ana@t.io", TenantID: tenant, CargoID: &cargoID, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	r := callsRouter(db, tenant, user.ID, "proprio")
	assert.Equal(t, http.StatusOK, hit(r, "/calls-receive"), "tem calls:receive")
	for _, a := range []string{"place", "read", "delete", "manage"} {
		assert.Equal(t, http.StatusForbidden, hit(r, "/calls-"+a), "sem calls:%s deve ser negado", a)
	}
}

func TestCallsPermissions_CargoWithoutAnyCallPermissionIsDeniedEverywhere(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	cargoID := seedCargo(t, db, tenant, "Atendente")
	seedPermission(t, db, "tickets", "read")
	user := models.User{Name: "Ana", Email: "ana@t.io", TenantID: tenant, CargoID: &cargoID, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	r := callsRouter(db, tenant, user.ID, "proprio")
	for _, a := range []string{"receive", "place", "read", "delete", "manage"} {
		assert.Equal(t, http.StatusForbidden, hit(r, "/calls-"+a))
	}
}

// Alcance de empresa passa sem cargo (comportamento herdado; consta na nota de
// atualização).
func TestCallsPermissions_TenantScopeBypassesCargo(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	user := models.User{Name: "Adm", Email: "adm@t.io", TenantID: tenant, Alcance: "tenant"}
	require.NoError(t, db.Create(&user).Error)

	r := callsRouter(db, tenant, user.ID, "tenant")
	for _, a := range []string{"receive", "place", "read", "delete", "manage"} {
		assert.Equal(t, http.StatusOK, hit(r, "/calls-"+a))
	}
}

func TestCallsPermissions_PermissionFromAnotherTenantsCargoDoesNotCount(t *testing.T) {
	db := testutil.NewTestDB(t)
	mine, other := uuid.New(), uuid.New()
	myCargo := seedCargo(t, db, mine, "Atendente")
	otherCargo := seedCargo(t, db, other, "Atendente")
	place := seedPermission(t, db, "calls", "place")
	require.NoError(t, db.Create(&models.CargoPermissao{CargoID: otherCargo, PermissionID: place}).Error)
	user := models.User{Name: "Ana", Email: "ana@t.io", TenantID: mine, CargoID: &myCargo, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	assert.Equal(t, http.StatusForbidden, hit(callsRouter(db, mine, user.ID, "proprio"), "/calls-place"))
}

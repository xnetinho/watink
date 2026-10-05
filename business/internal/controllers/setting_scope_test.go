package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Um broadcaster que usa o SSEHub REAL: o que importa aqui é quem de fato recebe
// o evento, não que uma função tenha sido chamada.
func realSSEBroadcaster() (*services.SSEHub, *services.SSEBroadcast) {
	hub := services.NewSSEHub()
	return hub, services.NewSSEBroadcast(hub)
}

func TestUpdateSetting_EventReachesOnlyTheOwnTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()

	hub, bc := realSSEBroadcaster()
	// Cada conexão SSE real se inscreve em "tenant:<id>" (controllers/sse.go).
	chA, cleanA := hub.Register("a", []string{"tenant:" + tenantA.String(), "notification"})
	defer cleanA()
	chB, cleanB := hub.Register("b", []string{"tenant:" + tenantB.String(), "notification"})
	defer cleanB()

	ctrl := NewSettingController(&mockSettingRepo{}, bc)
	payload, _ := json.Marshal(map[string]string{"value": "whatsapp"})
	c, w := setupSettingContext(t, db, tenantA, "PUT", "/settings/theme", payload)
	c.Params = gin.Params{{Key: "key", Value: "theme"}}
	ctrl.UpdateSetting(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	recebeu := func(ch <-chan string) bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	}
	assert.True(t, recebeu(chA), "o próprio tenant deve receber a atualização de setting em tempo real")
	assert.False(t, recebeu(chB), "o evento de setting de um tenant NÃO pode chegar a outro tenant")
}

func TestDeleteWhatsApp_EventReachesOnlyTheOwnTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub, bc := realSSEBroadcaster()
	tenantA, tenantB := uuid.New(), uuid.New()
	chA, cleanA := hub.Register("a", []string{"tenant:" + tenantA.String()})
	defer cleanA()
	chB, cleanB := hub.Register("b", []string{"tenant:" + tenantB.String()})
	defer cleanB()

	bc.EmitToTenantRoom(tenantA.String(), "whatsapp", gin.H{"action": "delete", "whatsappId": 1})

	select {
	case <-chA:
	default:
		t.Fatal("o próprio tenant deve receber")
	}
	select {
	case <-chB:
		t.Fatal("outro tenant não pode receber")
	default:
	}
}

// ── Gate de permissão ───────────────────────────────────────────────────────

func settingsRouter(t *testing.T, ctrl *SettingController, tenantID uuid.UUID, alcance string, userID int) *gin.Engine {
	t.Helper()
	db := testutil.NewTestDB(t)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID)
		c.Set("alcance", alcance)
		c.Set("userId", float64(userID))
		c.Next()
	})
	r.PUT("/settings/:key", auth.RequirePermission("settings", "update"), ctrl.UpdateSetting)
	// guarda o db para os testes semearem usuário/cargo
	r.GET("/_db", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func putSetting(r *gin.Engine, key, value string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"value": value})
	req := httptest.NewRequest(http.MethodPut, "/settings/"+key, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSettingsRoute_AtendenteWithoutPermissionIsDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	cargoID := seedCargo(t, db, tenantID, "Atendente")
	permID := seedPermission(t, db, "tickets", "read")
	require.NoError(t, db.Create(&models.CargoPermissao{CargoID: cargoID, PermissionID: permID}).Error)
	user := models.User{Name: "Ana", Email: "ana@t.io", TenantID: tenantID, CargoID: &cargoID, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID)
		c.Set("alcance", "proprio")
		c.Set("userId", float64(user.ID))
		c.Next()
	})
	r.PUT("/settings/:key", auth.RequirePermission("settings", "update"), NewSettingController(&mockSettingRepo{}, nil).UpdateSetting)

	w := putSetting(r, "systemTitle", "Hackeado")
	assert.Equal(t, http.StatusForbidden, w.Code, "atendente sem settings:update não pode gravar configuração do tenant")

	var n int64
	db.Model(&models.Setting{}).Where(`"tenantId" = ? AND key = ?`, tenantID, "systemTitle").Count(&n)
	assert.Zero(t, n, "nada pode ter sido gravado")
}

func TestSettingsRoute_CargoWithSettingsUpdateIsAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	cargoID := seedCargo(t, db, tenantID, "Admin")
	permID := seedPermission(t, db, "settings", "update")
	require.NoError(t, db.Create(&models.CargoPermissao{CargoID: cargoID, PermissionID: permID}).Error)
	user := models.User{Name: "Bia", Email: "bia@t.io", TenantID: tenantID, CargoID: &cargoID, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID)
		c.Set("alcance", "proprio")
		c.Set("userId", float64(user.ID))
		c.Next()
	})
	r.PUT("/settings/:key", auth.RequirePermission("settings", "update"), NewSettingController(&mockSettingRepo{}, nil).UpdateSetting)

	assert.Equal(t, http.StatusOK, putSetting(r, "systemTitle", "Minha Empresa").Code)
}

func TestSettingsRoute_AdminAlcanceTenantIsAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	ctrl := NewSettingController(&mockSettingRepo{}, nil)
	r := settingsRouter(t, ctrl, tenantID, "tenant", 1)
	assert.Equal(t, http.StatusOK, putSetting(r, "theme", "whatsapp").Code)
}

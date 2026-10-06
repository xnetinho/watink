package controllers

import (
	"encoding/json"
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

func TestNormalizeCallRecordingMode(t *testing.T) {
	for in, want := range map[string]string{
		"": "off", "off": "off", "OFF": "off", "optional": "optional", " Auto ": "auto", "auto": "auto",
		"on": "off", "true": "off", "lixo": "off", "1": "off",
	} {
		assert.Equal(t, want, NormalizeCallRecordingMode(in), "entrada %q", in)
	}
}

func TestCallRecordingModeOf_AbsentIsOff(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	assert.Equal(t, "off", callRecordingModeOf(db, tenant), "empresa que nunca configurou não grava")

	require.NoError(t, db.Create(&models.Setting{Key: CallRecordingModeKey, TenantID: tenant, Value: "auto"}).Error)
	assert.Equal(t, "auto", callRecordingModeOf(db, tenant))
	assert.Equal(t, "off", callRecordingModeOf(db, uuid.New()), "o modo de outra empresa não vaza")

	require.NoError(t, db.Model(&models.Setting{}).Where(`key = ? AND "tenantId" = ?`, CallRecordingModeKey, tenant).Update("value", "valor-corrompido").Error)
	assert.Equal(t, "off", callRecordingModeOf(db, tenant), "valor inválido nunca liga a gravação")
}

// 5.4: o PUT genérico de settings (só settings:update) NÃO muda a gravação de
// chamadas: sem a rota de aceite nada muda.
func TestUpdateSetting_CallRecordingKeysAreRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	cargoID := seedCargo(t, db, tenant, "Config")
	permID := seedPermission(t, db, "settings", "update")
	require.NoError(t, db.Create(&models.CargoPermissao{CargoID: cargoID, PermissionID: permID}).Error)
	user := models.User{Name: "X", Email: "x@t.io", TenantID: tenant, CargoID: &cargoID, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenant)
		c.Set("alcance", "proprio")
		c.Set("userId", float64(user.ID))
		c.Next()
	})
	r.PUT("/settings/:key", auth.RequirePermission("settings", "update"), NewSettingController(&mockSettingRepo{}, nil).UpdateSetting)

	for _, key := range []string{CallRecordingModeKey, callRecordingAckByKey, callRecordingAckAtKey} {
		w := putSetting(r, key, "auto")
		assert.Equal(t, http.StatusForbidden, w.Code, "%s não pode ser gravada pelo PUT genérico", key)
	}
	var n int64
	db.Model(&models.Setting{}).Where(`"tenantId" = ? AND key LIKE 'callRecording%'`, tenant).Count(&n)
	assert.Zero(t, n, "nada pode ter sido gravado")

	assert.Equal(t, http.StatusOK, putSetting(r, "systemTitle", "ok").Code, "as demais settings continuam funcionando")
}

func listAs(t *testing.T, db *gorm.DB, tenant uuid.UUID, userID int, alcance string) []models.Setting {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenant)
		c.Set("alcance", alcance)
		c.Set("userId", float64(userID))
		c.Next()
	})
	r.GET("/settings", NewSettingController(&mockSettingRepo{}, nil).ListSettings)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/settings", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var out []models.Setting
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

func keysOf(ss []models.Setting) map[string]string {
	m := map[string]string{}
	for _, s := range ss {
		m[s.Key] = s.Value
	}
	return m
}

// 5.5: quem não tem calls:manage não vê modo nem aceite; quem tem, vê.
func TestListSettings_CallRecordingOnlyForCallsManage(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	for _, s := range []models.Setting{
		{Key: CallRecordingModeKey, TenantID: tenant, Value: "auto"},
		{Key: callRecordingAckByKey, TenantID: tenant, Value: "7"},
		{Key: callRecordingAckAtKey, TenantID: tenant, Value: "2026-10-06T10:00:00Z"},
		{Key: "systemTitle", TenantID: tenant, Value: "Acme"},
	} {
		require.NoError(t, db.Create(&s).Error)
	}

	commonCargo := seedCargo(t, db, tenant, "Atendente")
	common := models.User{Name: "Ana", Email: "ana@t.io", TenantID: tenant, CargoID: &commonCargo, Alcance: "proprio"}
	require.NoError(t, db.Create(&common).Error)
	got := keysOf(listAs(t, db, tenant, common.ID, "proprio"))
	assert.Equal(t, "Acme", got["systemTitle"], "as demais settings seguem visíveis")
	for _, k := range []string{CallRecordingModeKey, callRecordingAckByKey, callRecordingAckAtKey} {
		_, has := got[k]
		assert.False(t, has, "%s não pode ser exposta sem calls:manage", k)
	}

	manageCargo := seedCargo(t, db, tenant, "Chamadas")
	managePerm := seedPermission(t, db, "calls", "manage")
	require.NoError(t, db.Create(&models.CargoPermissao{CargoID: manageCargo, PermissionID: managePerm}).Error)
	mgr := models.User{Name: "Bia", Email: "bia@t.io", TenantID: tenant, CargoID: &manageCargo, Alcance: "proprio"}
	require.NoError(t, db.Create(&mgr).Error)
	got = keysOf(listAs(t, db, tenant, mgr.ID, "proprio"))
	assert.Equal(t, "auto", got[CallRecordingModeKey], "calls:manage vê o modo")
	assert.Equal(t, "7", got[callRecordingAckByKey])

	admin := models.User{Name: "Adm", Email: "adm@t.io", TenantID: tenant, Alcance: "tenant"}
	require.NoError(t, db.Create(&admin).Error)
	assert.Equal(t, "auto", keysOf(listAs(t, db, tenant, admin.ID, "tenant"))[CallRecordingModeKey], "alcance de empresa vê")
}

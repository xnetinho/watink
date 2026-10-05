package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestIsSecretSettingKey(t *testing.T) {
	for _, k := range []string{"aiApiKey", "aiEmbeddingApiKey", "userApiToken", "smtpPassword", "webhookSecret", "AIAPIKEY", "integrationToken"} {
		assert.True(t, isSecretSettingKey(k), "%s deveria ser tratada como segredo", k)
	}
	for _, k := range []string{"aiModel", "aiProvider", "aiCustomBaseURL", "systemTitle", "theme", "language", "timezone", "aiEnabled", "tokenizer"} {
		assert.False(t, isSecretSettingKey(k), "%s NÃO é segredo", k)
	}
}

func TestMaskSecretSettings_KeepsEmptyAndDoesNotMutateInput(t *testing.T) {
	in := []models.Setting{
		{Key: "aiApiKey", Value: "sk-real-123"},
		{Key: "aiEmbeddingApiKey", Value: ""},
		{Key: "aiModel", Value: "gpt-4o"},
	}
	out := maskSecretSettings(in)

	assert.Equal(t, maskedSecretPlaceholder, out[0].Value, "chave configurada vira placeholder")
	assert.Equal(t, "", out[1].Value, "chave vazia continua vazia (UI distingue 'não configurado')")
	assert.Equal(t, "gpt-4o", out[2].Value, "setting comum não é alterada")
	assert.Equal(t, "sk-real-123", in[0].Value, "a lista original não pode ser mutada")
}

func seedSecretSettings(t *testing.T, db *gorm.DB, tenantID uuid.UUID) {
	t.Helper()
	for k, v := range map[string]string{"aiApiKey": "sk-real-123", "aiEmbeddingApiKey": "emb-real-456", "aiModel": "gpt-4o"} {
		require.NoError(t, db.Create(&models.Setting{Key: k, Value: v, TenantID: tenantID}).Error)
	}
}

func listSettingsAs(t *testing.T, db *gorm.DB, tenantID uuid.UUID, alcance string, userID int) map[string]string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/settings", nil)
	c.Set("tenantId", tenantID)
	c.Set("alcance", alcance)
	c.Set("userId", float64(userID))
	c.Set("db", db.Where(`"tenantId" = ?`, tenantID))

	NewSettingController(&mockSettingRepo{}, nil).ListSettings(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var rows []models.Setting
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rows))
	got := map[string]string{}
	for _, r := range rows {
		got[r.Key] = r.Value
	}
	return got
}

func TestListSettings_AtendenteSeesMaskedSecrets(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	seedSecretSettings(t, db, tenantID)

	cargoID := seedCargo(t, db, tenantID, "Atendente")
	permID := seedPermission(t, db, "tickets", "read")
	require.NoError(t, db.Create(&models.CargoPermissao{CargoID: cargoID, PermissionID: permID}).Error)
	user := models.User{Name: "Ana", Email: "ana@t.io", TenantID: tenantID, CargoID: &cargoID, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	got := listSettingsAs(t, db, tenantID, "proprio", user.ID)

	assert.Equal(t, maskedSecretPlaceholder, got["aiApiKey"], "atendente não pode ver a chave de API")
	assert.Equal(t, maskedSecretPlaceholder, got["aiEmbeddingApiKey"])
	assert.Equal(t, "gpt-4o", got["aiModel"], "o que não é segredo continua visível")
	assert.NotContains(t, got["aiApiKey"], "sk-real")
}

func TestListSettings_CargoWithSettingsUpdateSeesRealSecrets(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	seedSecretSettings(t, db, tenantID)

	cargoID := seedCargo(t, db, tenantID, "Admin")
	permID := seedPermission(t, db, "settings", "update")
	require.NoError(t, db.Create(&models.CargoPermissao{CargoID: cargoID, PermissionID: permID}).Error)
	user := models.User{Name: "Bia", Email: "bia@t.io", TenantID: tenantID, CargoID: &cargoID, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	got := listSettingsAs(t, db, tenantID, "proprio", user.ID)
	assert.Equal(t, "sk-real-123", got["aiApiKey"], "quem tem settings:update vê a chave real")
	assert.Equal(t, "emb-real-456", got["aiEmbeddingApiKey"])
}

func TestListSettings_AlcanceTenantSeesRealSecrets(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	seedSecretSettings(t, db, tenantID)

	got := listSettingsAs(t, db, tenantID, "tenant", 1)
	assert.Equal(t, "sk-real-123", got["aiApiKey"], "administrador (alcance tenant) vê a chave real")
}

func TestListSettings_UnknownUserFailsClosedToMasked(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	seedSecretSettings(t, db, tenantID)

	got := listSettingsAs(t, db, tenantID, "proprio", 999999)
	assert.Equal(t, maskedSecretPlaceholder, got["aiApiKey"], "na dúvida, mascara (fail-closed)")
}

func TestListSettings_SecretsOfOtherTenantNeverAppear(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	seedSecretSettings(t, db, tenantB)

	got := listSettingsAs(t, db, tenantA, "tenant", 1)
	assert.Empty(t, got, "tenant A não enxerga settings do tenant B")
}

func TestUpdateSetting_BroadcastNeverCarriesSecretValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	hub, bc := realSSEBroadcaster()
	ch, clean := hub.Register("a", []string{"tenant:" + tenantID.String()})
	defer clean()

	payload, _ := json.Marshal(map[string]string{"value": "sk-super-secreto"})
	c, w := setupSettingContext(t, db, tenantID, "PUT", "/settings/aiApiKey", payload)
	c.Params = gin.Params{{Key: "key", Value: "aiApiKey"}}
	NewSettingController(&mockSettingRepo{}, bc).UpdateSetting(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	select {
	case msg := <-ch:
		assert.NotContains(t, msg, "sk-super-secreto", "o evento vai a todo o tenant: não pode carregar o segredo")
		assert.Contains(t, msg, maskedSecretPlaceholder)
	default:
		t.Fatal("o tenant deveria receber o evento")
	}
	assert.Contains(t, w.Body.String(), "sk-super-secreto", "quem gravou (já tem settings:update) recebe a resposta normal")
}

package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/plugins"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func pluginRouter(t *testing.T, db *gorm.DB, tenantID uuid.UUID) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctrl := NewPluginController(nil, db, plugins.NewPluginRegistry(db))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID.String())
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.GET("/plugins/installed", ctrl.Installed)
	r.POST("/plugins/:slug/activate", ctrl.Activate)
	r.POST("/plugins/:slug/deactivate", ctrl.Deactivate)
	return r
}

func callPlugin(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func installed(t *testing.T, r *gin.Engine) (active []string, statuses map[string]string) {
	t.Helper()
	w := callPlugin(r, http.MethodGet, "/plugins/installed")
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Active   []string          `json:"active"`
		Statuses map[string]string `json:"statuses"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Active, body.Statuses
}

func TestPlugin_AtivarLiberaOGateSemLicenca(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	r := pluginRouter(t, db, tenant)
	reg := plugins.NewPluginRegistry(db)

	// Embarcados são "pro" no manifesto, mas não existe mais licença: ativar basta.
	for _, slug := range plugins.EmbeddedSlugs() {
		assert.Equal(t, "blocked", string(reg.GetStatus(tenant, slug)), "%s começa bloqueado (não alocado)", slug)
		w := callPlugin(r, http.MethodPost, "/plugins/"+slug+"/activate")
		require.Equal(t, http.StatusOK, w.Code, "ativar %s sem licença", slug)
		assert.Equal(t, "active", string(reg.GetStatus(tenant, slug)), "%s fica ativo depois de alocado", slug)
	}

	active, statuses := installed(t, r)
	assert.ElementsMatch(t, plugins.EmbeddedSlugs(), active)
	for _, slug := range plugins.EmbeddedSlugs() {
		assert.Equal(t, "active", statuses[slug])
	}
}

func TestPlugin_DesativarBloqueiaEPreservaHistorico(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	r := pluginRouter(t, db, tenant)
	reg := plugins.NewPluginRegistry(db)

	require.Equal(t, http.StatusOK, callPlugin(r, http.MethodPost, "/plugins/helpdesk/activate").Code)
	require.Equal(t, http.StatusOK, callPlugin(r, http.MethodPost, "/plugins/helpdesk/deactivate").Code)
	assert.Equal(t, "blocked", string(reg.GetStatus(tenant, "helpdesk")))

	var n int64
	require.NoError(t, db.Model(&models.PluginInstallation{}).Where(`"tenantId" = ? AND "pluginId" = ?`, tenant, "helpdesk").Count(&n).Error)
	assert.EqualValues(t, 1, n, "desativar nunca apaga a linha (auditoria)")

	// Reativar reaproveita a linha, sem duplicar.
	require.Equal(t, http.StatusOK, callPlugin(r, http.MethodPost, "/plugins/helpdesk/activate").Code)
	require.NoError(t, db.Model(&models.PluginInstallation{}).Where(`"tenantId" = ? AND "pluginId" = ?`, tenant, "helpdesk").Count(&n).Error)
	assert.EqualValues(t, 1, n)
	assert.Equal(t, "active", string(reg.GetStatus(tenant, "helpdesk")))
}

func TestPlugin_SlugDesconhecidoNaoAtiva(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	r := pluginRouter(t, db, tenant)

	w := callPlugin(r, http.MethodPost, "/plugins/plugin-de-terceiros/activate")
	assert.Equal(t, http.StatusNotFound, w.Code, "só plugins embarcados podem ser ativados")

	var n int64
	require.NoError(t, db.Model(&models.PluginInstallation{}).Where(`"tenantId" = ?`, tenant).Count(&n).Error)
	assert.Zero(t, n, "nada é gravado para slug desconhecido")
}

func TestPlugin_AlocacaoEPorEmpresa(t *testing.T) {
	db := testutil.NewTestDB(t)
	a, b := uuid.New(), uuid.New()
	ra, rb := pluginRouter(t, db, a), pluginRouter(t, db, b)
	reg := plugins.NewPluginRegistry(db)

	require.Equal(t, http.StatusOK, callPlugin(ra, http.MethodPost, "/plugins/groups/activate").Code)

	assert.Equal(t, "active", string(reg.GetStatus(a, "groups")))
	assert.Equal(t, "blocked", string(reg.GetStatus(b, "groups")), "ativar na empresa A não libera a B")

	activeB, _ := installed(t, rb)
	assert.Empty(t, activeB)

	// A empresa B desativar o mesmo slug não derruba a A.
	require.Equal(t, http.StatusOK, callPlugin(rb, http.MethodPost, "/plugins/groups/deactivate").Code)
	assert.Equal(t, "active", string(reg.GetStatus(a, "groups")))
}

func TestPlugin_EmbeddedSlugsCobreOsCincoPlugins(t *testing.T) {
	assert.ElementsMatch(t, []string{"helpdesk", "webchat", "assistant", "groups", "inventory-advanced"}, plugins.EmbeddedSlugs())
	assert.True(t, plugins.IsEmbeddedSlug("helpdesk"))
	assert.False(t, plugins.IsEmbeddedSlug("outro"))
	assert.False(t, plugins.IsEmbeddedSlug(""))
}

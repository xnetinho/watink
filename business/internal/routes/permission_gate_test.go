package routes

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/application"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/plugins"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var writeMethods = map[string]bool{
	http.MethodPost: true, http.MethodPut: true, http.MethodPatch: true, http.MethodDelete: true,
}

func skippedPrefix(path string) bool {
	for p := range permissionRoutePrefixSkip {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func requiresUserJWT(r *gin.Engine, method, path string) bool {
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	w := httptest.NewRecorder()
	func() {
		defer func() { _ = recover() }()
		r.ServeHTTP(w, req)
	}()
	return w.Code == http.StatusUnauthorized && strings.Contains(w.Body.String(), "Authorization header required")
}

func routeKey(method, path string) string { return method + " " + path }

// concretePath troca os :parâmetros por valores qualquer para chamar a rota.
func concretePath(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "1"
		} else if strings.HasPrefix(p, "*") {
			parts[i] = "x"
		}
	}
	return strings.Join(parts, "/")
}

func serve(r *gin.Engine, method, path, token string) int {
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	func() {
		defer func() { _ = recover() }() // handler sem dependências do teste pode entrar em pânico: isso NÃO é 403
		r.ServeHTTP(w, req)
	}()
	if w.Code == 0 {
		return http.StatusInternalServerError
	}
	return w.Code
}

// Percorre as rotas REAIS (SetupRoutes). Para cada rota autenticada de escrita, chama com um usuário de alcance
// "proprio" cujo cargo NÃO tem nenhuma permissão. Rota gateada responde 403; qualquer outra resposta significa que o
// handler foi alcançado sem permissão. Isenção só existe em permissionExemptions, com justificativa.
func TestEveryAuthenticatedWriteRouteRequiresPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "Empresa Gate", Status: "active"}).Error)

	cargo := models.Cargo{Name: "Sem permissao", TenantID: tenantID}
	require.NoError(t, db.Create(&cargo).Error)
	user := models.User{Name: "Sem Permissao", Email: "sem@perm.io", TenantID: tenantID, CargoID: &cargo.ID, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	// Plugins só respondem à permissão quando estão ativos para a empresa; inativos negam antes, com 402.
	for _, slug := range plugins.EmbeddedSlugs() {
		require.NoError(t, db.Create(&models.PluginInstallation{TenantID: tenantID, PluginID: slug, Active: true}).Error)
	}

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": user.ID, "tenantId": tenantID.String(), "alcance": "proprio", "username": user.Name, "exp": 4102444800,
	}).SignedString([]byte("test-secret"))
	require.NoError(t, err)

	r := gin.New()
	SetupRoutes(r.Group("/api/v1"), nil, application.NewContainer(db, nil, nil, nil), nil, nil, nil, nil)

	var unprotected []string
	seenExemptions := map[string]bool{}
	checked := 0
	for _, ri := range r.Routes() {
		if !writeMethods[ri.Method] || skippedPrefix(ri.Path) {
			continue
		}
		path := concretePath(ri.Path)
		// Rota autenticada pelo JWT de usuário: sem token responde 401 com a mensagem do IsAuth. Rotas públicas
		// (webhook, login, refresh_token) não passam por ele; o refresh também responde 401, mas por outro motivo.
		if !requiresUserJWT(r, ri.Method, path) {
			continue
		}
		key := routeKey(ri.Method, ri.Path)
		code := serve(r, ri.Method, path, tok)
		checked++
		if _, exempt := permissionExemptions[key]; exempt {
			seenExemptions[key] = true
			continue
		}
		if code != http.StatusForbidden {
			unprotected = append(unprotected, key+" -> "+http.StatusText(code))
		}
	}

	require.Greater(t, checked, 100, "o teste precisa ter percorrido as rotas de escrita reais")
	sort.Strings(unprotected)
	assert.Empty(t, unprotected, "rotas autenticadas de escrita alcançadas sem permissão (adicione auth.RequirePermission ou uma isenção justificada)")

	for key := range permissionExemptions {
		assert.True(t, seenExemptions[key], "isenção velha: a rota %q não existe mais como rota autenticada de escrita", key)
	}
}

func TestPermissionExemptions_TemJustificativa(t *testing.T) {
	for key, why := range permissionExemptions {
		assert.GreaterOrEqual(t, len(strings.TrimSpace(why)), 15, "isenção %q sem justificativa suficiente", key)
	}
	for prefix, why := range permissionRoutePrefixSkip {
		assert.GreaterOrEqual(t, len(strings.TrimSpace(why)), 15, "prefixo ignorado %q sem justificativa suficiente", prefix)
	}
}

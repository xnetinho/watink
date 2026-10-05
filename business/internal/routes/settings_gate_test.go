package routes

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/application"
	"github.com/alltomatos/watinkdev/business/internal/controllers"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Monta as rotas REAIS (SetupRoutes) e bate no PUT /settings/:key com um JWT de
// atendente. Se alguém tirar o RequirePermission do routes.go, este teste
// falha — ao contrário de um teste que monta a própria rota.
func TestPutSettings_RealRouteDeniesUserWithoutSettingsUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()

	cargo := models.Cargo{Name: "Atendente", TenantID: tenantID}
	require.NoError(t, db.Create(&cargo).Error)
	user := models.User{Name: "Ana", Email: "ana@t.io", TenantID: tenantID, CargoID: &cargo.ID, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": user.ID, "tenantId": tenantID.String(), "alcance": "proprio", "username": "Ana",
		"exp": 4102444800,
	}).SignedString([]byte("test-secret"))
	require.NoError(t, err)

	r := gin.New()
	SetupRoutes(r.Group("/api/v1"), nil, application.NewContainer(db, nil, nil, nil), nil, controllers.BuildInfo{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/systemTitle", bytes.NewReader([]byte(`{"value":"Hackeado"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "rota real: atendente sem settings:update deve tomar 403 — %s", w.Body.String())

	var n int64
	db.Model(&models.Setting{}).Where(`"tenantId" = ? AND key = ?`, tenantID, "systemTitle").Count(&n)
	assert.Zero(t, n, "nada pode ter sido gravado")
}

package routes

import (
	"net/http"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/application"
	"github.com/alltomatos/watinkdev/business/internal/database"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// O cargo Atendente que o InitializeTenant cria de verdade precisa continuar fazendo o trabalho do dia a dia, e só ele:
// gateando as mutações não pode trancar o atendimento nem abrir a gestão.
func TestAtendentePadrao_FazODiaADiaEFicaForaDaGestao(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")
	db := testutil.NewTestDB(t)
	database.DB = db
	database.Seed()

	require.NoError(t, services.NewSetupService(db).InitializeTenant(services.TenantSeedData{
		CompanyName: "Empresa", FirstName: "Adm", LastName: "In", Email: "adm@e.io", Password: "secret123",
	}))
	var admin models.User
	require.NoError(t, db.Where("email = ?", "adm@e.io").First(&admin).Error)
	var cargo models.Cargo
	require.NoError(t, db.Where(`name = ? AND "tenantId" = ?`, "Atendente", admin.TenantID).First(&cargo).Error)
	agent := models.User{Name: "Ana", Email: "ana@e.io", TenantID: admin.TenantID, CargoID: &cargo.ID, Alcance: "proprio"}
	require.NoError(t, db.Create(&agent).Error)

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": agent.ID, "tenantId": admin.TenantID.String(), "alcance": "proprio", "username": "Ana", "exp": 4102444800,
	}).SignedString([]byte("test-secret"))
	require.NoError(t, err)

	r := gin.New()
	SetupRoutes(r.Group("/api/v1"), nil, application.NewContainer(db, nil, nil, nil), nil, nil, nil, nil)

	// Atendimento: o gate NÃO barra (qualquer resposta que não seja 403).
	for _, route := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/messages/1"},
		{http.MethodPost, "/api/v1/contacts"},
		{http.MethodPut, "/api/v1/contacts/1"},
		{http.MethodPost, "/api/v1/message/x/react"},
		{http.MethodPost, "/api/v1/tags"},
		{http.MethodPut, "/api/v1/tags/1"},
	} {
		assert.NotEqual(t, http.StatusForbidden, serve(r, route.method, route.path, tok), "atendente deve poder %s %s", route.method, route.path)
	}

	// Gestão: o gate barra.
	for _, route := range []struct{ method, path string }{
		{http.MethodDelete, "/api/v1/contacts/all"},
		{http.MethodPost, "/api/v1/contacts/bulk-delete"},
		{http.MethodPost, "/api/v1/queue"},
		{http.MethodDelete, "/api/v1/queue/1"},
		{http.MethodPost, "/api/v1/flows"},
		{http.MethodPost, "/api/v1/pipelines"},
		{http.MethodPost, "/api/v1/quickAnswers"},
		{http.MethodDelete, "/api/v1/tags/1"},
		{http.MethodPost, "/api/v1/plugins/helpdesk/activate"},
	} {
		assert.Equal(t, http.StatusForbidden, serve(r, route.method, route.path, tok), "atendente não pode %s %s", route.method, route.path)
	}
}

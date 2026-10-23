package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Caminho real completo: token da empresa → IsAuth → auth.GetScoped → consulta. É o que o SET LOCAL
// "protegia" sem nunca funcionar (e o usuário postgres ignora RLS): o isolamento é o filtro "tenantId" explícito.
// Estes testes garantem que ele não dependia do SET e travam uma regressão do filtro.
func TestIsolation_JWTToScopedQuery_OneCompanyNeverSeesTheOther(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret-isolation"
	require.NoError(t, os.Setenv("JWT_SECRET", secret))

	db := makeTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	seed := func(tenant uuid.UUID, ticket int, body string) {
		require.NoError(t, db.Create(&models.Message{ID: uuid.NewString(), Body: body, TicketID: ticket, TenantID: tenant}).Error)
	}
	seed(tenantA, 1, "mensagem da empresa A")
	seed(tenantB, 1, "mensagem da empresa B (mesmo ticketId)")

	r := gin.New()
	r.Use(IsAuth(db))
	r.GET("/messages", func(c *gin.Context) {
		scoped, _, ok := auth.GetScoped(c, "Messages")
		if !ok {
			return
		}
		var out []models.Message
		require.NoError(t, scoped.Where(`"ticketId" = ?`, 1).Find(&out).Error)
		c.JSON(http.StatusOK, out)
	})

	get := func(tenant uuid.UUID) []models.Message {
		signed := makeValidToken(t, secret, jwt.MapClaims{"tenantId": tenant.String(), "alcance": "tenant"})
		req, _ := http.NewRequest("GET", "/messages", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		return decodeMessages(t, w.Body.Bytes())
	}

	a := get(tenantA)
	require.Len(t, a, 1)
	assert.Equal(t, "mensagem da empresa A", a[0].Body)
	assert.Equal(t, tenantA, a[0].TenantID)

	b := get(tenantB)
	require.Len(t, b, 1)
	assert.Equal(t, "mensagem da empresa B (mesmo ticketId)", b[0].Body)
	assert.Equal(t, tenantB, b[0].TenantID)
}

// O handle injetado não pode ser "global ao pool" por acidente: duas requisições de empresas diferentes em
// sequência não podem se contaminar (o SET LOCAL, se funcionasse, também vazaria entre conexões do pool).
func TestIsolation_SequentialRequestsDoNotBleed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret-bleed"
	require.NoError(t, os.Setenv("JWT_SECRET", secret))

	db := makeTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	for i, tn := range []uuid.UUID{tenantA, tenantB} {
		require.NoError(t, db.Create(&models.Ticket{TenantID: tn, Status: "open", LastMessage: []string{"A", "B"}[i]}).Error)
	}

	r := gin.New()
	r.Use(IsAuth(db))
	r.GET("/tickets", func(c *gin.Context) {
		scoped, _, ok := auth.GetScoped(c, "Tickets")
		if !ok {
			return
		}
		var n int64
		scoped.Model(&models.Ticket{}).Count(&n)
		c.JSON(http.StatusOK, gin.H{"n": n})
	})
	for i := 0; i < 6; i++ {
		tn := []uuid.UUID{tenantA, tenantB}[i%2]
		signed := makeValidToken(t, secret, jwt.MapClaims{"tenantId": tn.String(), "alcance": "tenant"})
		req, _ := http.NewRequest("GET", "/tickets", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"n":1}`, w.Body.String(), "requisição %d da empresa %s viu linhas de outra empresa", i, tn)
	}
}

func decodeMessages(t *testing.T, b []byte) []models.Message {
	t.Helper()
	var out []models.Message
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

// O ramo "default" de GetScopedDB (qualquer tabela que não seja Tickets/Contacts) vale para um atendente comum,
// sem alcance de empresa: é o caminho da maioria das rotas e também tem de filtrar por empresa.
func TestIsolation_AgentDefaultBranchStillFiltersByCompany(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret-agent-default"
	require.NoError(t, os.Setenv("JWT_SECRET", secret))

	db := makeTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&models.Message{ID: uuid.NewString(), Body: "A", TicketID: 7, TenantID: tenantA}).Error)
	require.NoError(t, db.Create(&models.Message{ID: uuid.NewString(), Body: "B", TicketID: 7, TenantID: tenantB}).Error)

	r := gin.New()
	r.Use(IsAuth(db))
	r.GET("/messages", func(c *gin.Context) {
		scoped, _, ok := auth.GetScoped(c, "Messages")
		if !ok {
			return
		}
		var out []models.Message
		require.NoError(t, scoped.Find(&out).Error)
		c.JSON(http.StatusOK, out)
	})

	signed := makeValidToken(t, secret, jwt.MapClaims{"tenantId": tenantA.String(), "alcance": "proprio"})
	req, _ := http.NewRequest("GET", "/messages", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	got := decodeMessages(t, w.Body.Bytes())
	require.Len(t, got, 1, "atendente da empresa A viu %d linhas", len(got))
	assert.Equal(t, tenantA, got[0].TenantID)
}

// O ramo de atendente em Tickets: além da empresa, só os tickets dele; nunca os de outra empresa.
func TestIsolation_AgentTicketsBranchNeverCrossesCompanies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret-agent-tickets"
	require.NoError(t, os.Setenv("JWT_SECRET", secret))

	db := makeTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	userID := 42
	require.NoError(t, db.Create(&models.Ticket{TenantID: tenantA, Status: "open", UserID: &userID}).Error)
	require.NoError(t, db.Create(&models.Ticket{TenantID: tenantB, Status: "open", UserID: &userID}).Error)

	r := gin.New()
	r.Use(IsAuth(db))
	r.GET("/tickets", func(c *gin.Context) {
		scoped, _, ok := auth.GetScoped(c, "Tickets")
		if !ok {
			return
		}
		var out []models.Ticket
		require.NoError(t, scoped.Find(&out).Error)
		c.JSON(http.StatusOK, out)
	})

	signed := makeValidToken(t, secret, jwt.MapClaims{"tenantId": tenantA.String(), "alcance": "proprio", "id": float64(userID)})
	req, _ := http.NewRequest("GET", "/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got []models.Ticket
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Len(t, got, 1, "o atendente (mesmo userId nas duas empresas) viu %d tickets", len(got))
	assert.Equal(t, tenantA, got[0].TenantID)
}

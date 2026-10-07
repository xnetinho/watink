package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// sqlRecorder registra toda instrução SQL que o handle executa, sem depender do formato do log.
type sqlRecorder struct {
	mu   sync.Mutex
	list []string
}

func (r *sqlRecorder) attach(t *testing.T, db *gorm.DB) {
	t.Helper()
	cb := func(tx *gorm.DB) {
		r.mu.Lock()
		r.list = append(r.list, tx.Statement.SQL.String())
		r.mu.Unlock()
	}
	require.NoError(t, db.Callback().Raw().After("gorm:raw").Register("test:record_raw", cb))
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:record_query", cb))
	require.NoError(t, db.Callback().Row().After("gorm:row").Register("test:record_row", cb))
}

func (r *sqlRecorder) all() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.list...)
}

// O IsAuth não pode tocar o banco. O "SET LOCAL app.current_tenant = ?" que ele rodava nunca funcionou
// (SET não aceita parâmetro; fora de transação; e o usuário postgres ignora RLS), mas custava 1 round-trip
// em 100% das rotas protegidas e imprimia 'syntax error at or near "$1"' no log a cada requisição.
func TestIsAuth_ExecutesNoSQL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret-nosql"
	require.NoError(t, os.Setenv("JWT_SECRET", secret))

	db := makeTestDB(t)
	rec := &sqlRecorder{}
	rec.attach(t, db)

	tenantID := uuid.New().String()
	signed := makeValidToken(t, secret, jwt.MapClaims{"tenantId": tenantID})

	r := gin.New()
	r.Use(IsAuth(db))
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req, _ := http.NewRequest("GET", "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, rec.all(), "IsAuth não deve executar SQL; executou: %v", rec.all())
	for _, q := range rec.all() {
		assert.False(t, strings.Contains(strings.ToLower(q), "current_tenant"), "não deve mexer em app.current_tenant: %s", q)
	}
}

// O handle entregue aos controllers é o mesmo *gorm.DB injetado: nada de clone com estado escondido.
func TestIsAuth_InjectsTheSameDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret-samedb"
	require.NoError(t, os.Setenv("JWT_SECRET", secret))
	db := makeTestDB(t)
	signed := makeValidToken(t, secret, jwt.MapClaims{"tenantId": uuid.New().String()})

	var got *gorm.DB
	r := gin.New()
	r.Use(IsAuth(db))
	r.GET("/ping", func(c *gin.Context) {
		v, _ := c.Get("db")
		got, _ = v.(*gorm.DB)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	req, _ := http.NewRequest("GET", "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, got)
	assert.Same(t, db, got)
}

// A validação do UUID do token continua valendo (o tenantId alimenta todos os filtros "tenantId").
func TestIsAuth_StillRejectsNonUUIDTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret-uuid"
	require.NoError(t, os.Setenv("JWT_SECRET", secret))
	db := makeTestDB(t)
	signed := makeValidToken(t, secret, jwt.MapClaims{"tenantId": "x'; DROP TABLE \"Users\"; --"})

	r := gin.New()
	r.Use(IsAuth(db))
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	req, _ := http.NewRequest("GET", "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

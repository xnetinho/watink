package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Repositório GORM REAL: o que está em teste é a constraint do banco traduzida
// em 409, e um mock não passaria por ela.
func contactRouter(t *testing.T, db *gorm.DB, tenantID uuid.UUID) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctrl := NewContactController(repository.NewGORMContactRepo(db), nil, nil, nil)
	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/contacts", ctrl.CreateContact)
	r.PUT("/contacts/:contactId", ctrl.UpdateContact)
	return r
}

// conn cria uma conexão WhatsApp da empresa e devolve o id (todo contato pertence a uma conexão).
func conn(t *testing.T, db *gorm.DB, tenant uuid.UUID, name string) int {
	t.Helper()
	w := models.Whatsapp{Name: name, Status: "CONNECTED", TenantID: tenant}
	require.NoError(t, db.Create(&w).Error)
	return w.ID
}

func doJSON(r *gin.Engine, method, path string, payload map[string]interface{}) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateContact_DuplicateNumberInSameTenantIs409NotTheFormer500(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	r := contactRouter(t, db, tenant)
	c1 := conn(t, db, tenant, "Comercial")

	first := doJSON(r, http.MethodPost, "/contacts", map[string]interface{}{"name": "A", "number": "5511900000001", "connectionId": c1})
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())

	dup := doJSON(r, http.MethodPost, "/contacts", map[string]interface{}{"name": "B", "number": "5511900000001", "connectionId": c1})
	assert.Equal(t, http.StatusConflict, dup.Code, dup.Body.String())
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(dup.Body.Bytes(), &out))
	assert.Equal(t, "CONTACT_EXISTS", out["code"])
}

func TestCreateContact_SameNumberInAnotherTenantIsAllowed(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()

	a := doJSON(contactRouter(t, db, tenantA), http.MethodPost, "/contacts", map[string]interface{}{"name": "Cliente", "number": "5511900000002", "connectionId": conn(t, db, tenantA, "A")})
	require.Equal(t, http.StatusOK, a.Code, a.Body.String())

	b := doJSON(contactRouter(t, db, tenantB), http.MethodPost, "/contacts", map[string]interface{}{"name": "Cliente", "number": "5511900000002", "connectionId": conn(t, db, tenantB, "B")})
	assert.Equal(t, http.StatusOK, b.Code, "o mesmo contato em outra empresa é legítimo (multi-tenant): %s", b.Body.String())

	var n int64
	db.Model(&models.Contact{}).Where("number = ?", "5511900000002").Count(&n)
	assert.Equal(t, int64(2), n)
}

func TestUpdateContact_ChangingToAnExistingNumberIs409(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	r := contactRouter(t, db, tenant)
	c1 := conn(t, db, tenant, "Comercial")

	require.Equal(t, http.StatusOK, doJSON(r, http.MethodPost, "/contacts", map[string]interface{}{"name": "A", "number": "5511900000003", "connectionId": c1}).Code)
	second := doJSON(r, http.MethodPost, "/contacts", map[string]interface{}{"name": "B", "number": "5511900000004", "connectionId": c1})
	require.Equal(t, http.StatusOK, second.Code, second.Body.String())
	var created models.Contact
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &created))
	require.NotZero(t, created.ID)

	w := doJSON(r, http.MethodPut, "/contacts/"+strconv.Itoa(created.ID), map[string]interface{}{"number": "5511900000003"})
	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}

func TestCreateContact_MesmoNumeroEmDuasConexoesEhPermitidoNaMesmaNao(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	r := contactRouter(t, db, tenant)
	c1, c2 := conn(t, db, tenant, "Comercial"), conn(t, db, tenant, "Suporte")
	body := func(c int) map[string]interface{} {
		return map[string]interface{}{"name": "Cliente", "number": "5511900000009", "connectionId": c}
	}

	require.Equal(t, http.StatusOK, doJSON(r, http.MethodPost, "/contacts", body(c1)).Code)
	assert.Equal(t, http.StatusOK, doJSON(r, http.MethodPost, "/contacts", body(c2)).Code, "outra conexão = outro contato (cada um com seu histórico)")
	assert.Equal(t, http.StatusConflict, doJSON(r, http.MethodPost, "/contacts", body(c1)).Code, "na mesma conexão continua duplicado")
}

func TestLIDUniqueIndexIsPerTenant(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	lid := "777000111@lid"
	require.NoError(t, db.Create(&models.Contact{Name: "A", Number: "111", TenantID: tenantA, Lid: &lid}).Error)
	assert.NoError(t, db.Create(&models.Contact{Name: "B", Number: "222", TenantID: tenantB, Lid: &lid}).Error, "mesmo LID em outro tenant é permitido")
	assert.Error(t, db.Create(&models.Contact{Name: "C", Number: "333", TenantID: tenantA, Lid: &lid}).Error, "mesmo LID no mesmo tenant é rejeitado")
}

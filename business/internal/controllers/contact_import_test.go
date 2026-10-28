package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func importRouter(t *testing.T, db *gorm.DB, tenantID uuid.UUID) (*gin.Engine, *mockPublisher) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	pub := &mockPublisher{}
	ctrl := NewContactController(&MockContactRepo{db: db}, nil, pub, nil)
	r := gin.New()
	r.Use(testScopedMiddleware(db, tenantID.String()))
	r.POST("/contacts/import", ctrl.ImportContacts)
	return r, pub
}

func postImport(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/contacts/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func newWhatsapp(t *testing.T, db *gorm.DB, tenantID uuid.UUID, name, status string) models.Whatsapp {
	t.Helper()
	w := models.Whatsapp{Name: name, Status: status, TenantID: tenantID}
	require.NoError(t, db.Create(&w).Error)
	return w
}

func TestImportContacts_ExigeAConexao(t *testing.T) {
	db, tenantID := setupContactTestDB(t)
	r, pub := importRouter(t, db, tenantID)
	newWhatsapp(t, db, tenantID, "A", "CONNECTED")

	assert.Equal(t, http.StatusUnprocessableEntity, postImport(r, `{}`).Code, "sem connectionId não escolhe uma conexão por conta própria")
	assert.Equal(t, http.StatusUnprocessableEntity, postImport(r, `{"connectionId":9999}`).Code)
	assert.False(t, pub.called)
}

func TestImportContacts_ConexaoDesconectadaRecusa(t *testing.T) {
	db, tenantID := setupContactTestDB(t)
	r, pub := importRouter(t, db, tenantID)
	w := newWhatsapp(t, db, tenantID, "A", "DISCONNECTED")

	assert.Equal(t, http.StatusConflict, postImport(r, `{"connectionId":`+strconv.Itoa(w.ID)+`}`).Code)
	assert.False(t, pub.called)
}

func TestImportContacts_PublicaParaAConexaoEscolhidaMesmoComOutraConectada(t *testing.T) {
	db, tenantID := setupContactTestDB(t)
	r, pub := importRouter(t, db, tenantID)
	newWhatsapp(t, db, tenantID, "Primeira", "CONNECTED")
	second := newWhatsapp(t, db, tenantID, "Segunda", "CONNECTED")

	w := postImport(r, `{"connectionId":`+strconv.Itoa(second.ID)+`}`)
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	assert.Equal(t, "wbot."+tenantID.String()+"."+strconv.Itoa(second.ID)+".contact.import", pub.routingKey, "antes ia sempre para a primeira conectada")
}

func TestImportContacts_ConexaoDeOutraEmpresaNaoEhAceita(t *testing.T) {
	db, tenantID := setupContactTestDB(t)
	r, pub := importRouter(t, db, tenantID)
	alheia := newWhatsapp(t, db, uuid.New(), "Alheia", "CONNECTED")

	assert.Equal(t, http.StatusUnprocessableEntity, postImport(r, `{"connectionId":`+strconv.Itoa(alheia.ID)+`}`).Code)
	assert.False(t, pub.called)
}

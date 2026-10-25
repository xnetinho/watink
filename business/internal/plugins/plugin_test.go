package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/alltomatos/watinkdev/business/pkg/sdk"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// ---- MockWatinkCore implementa sdk.WatinkCore completa ----

type MockWatinkCore struct {
	mock.Mock
	registeredRoutes []registeredRoute
}

type registeredRoute struct {
	Method  string
	Path    string
	Handler gin.HandlerFunc
}

func (m *MockWatinkCore) GetDB() *gorm.DB {
	args := m.Called()
	if db, ok := args.Get(0).(*gorm.DB); ok {
		return db
	}
	return nil
}

func (m *MockWatinkCore) RegisterRoute(method string, path string, handler gin.HandlerFunc) {
	m.Called(method, path, handler)
	m.registeredRoutes = append(m.registeredRoutes, registeredRoute{
		Method: method, Path: path, Handler: handler,
	})
}

func (m *MockWatinkCore) RegisterPublicRoute(method string, path string, handler gin.HandlerFunc) {
	m.Called(method, path, handler)
	m.registeredRoutes = append(m.registeredRoutes, registeredRoute{
		Method: method, Path: path, Handler: handler,
	})
}

func (m *MockWatinkCore) EmitSocketEvent(room string, event string, payload interface{}) {
	m.Called(room, event, payload)
}

func (m *MockWatinkCore) GetStatus() sdk.PluginStatus {
	args := m.Called()
	return args.Get(0).(sdk.PluginStatus)
}

func (m *MockWatinkCore) SendTicketMessage(tenantID uuid.UUID, ticketID int, body string) error {
	args := m.Called(tenantID, ticketID, body)
	return args.Error(0)
}

// ---- Helpers ----

func setupPluginTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewTestDB(t)
}

// authedRouter simula o que IsAuth injeta: o db, a empresa e o alcance "tenant" (que passa em RequirePermission).
func authedRouter(db *gorm.DB, tenantID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("tenantId", tenantID.String())
		c.Set("alcance", "tenant")
		c.Next()
	})
	return r
}

// ---- WebchatPlugin Tests ----

// TestWebchatPlugin_GetManifest verifica metadata do plugin
func TestWebchatPlugin_GetManifest(t *testing.T) {
	p := &WebchatPlugin{}
	m := p.GetManifest()
	assert.Equal(t, "webchat", m.Slug)
	assert.Equal(t, "1.0.0", m.Version)
	assert.Equal(t, "pro", m.Type)
}

// TestWebchatPlugin_OnActivate_RegistersRoutes verifica que 2 rotas são registradas
func TestWebchatPlugin_OnActivate_RegistersRoutes(t *testing.T) {
	db := setupPluginTestDB(t)
	mockCore := new(MockWatinkCore)
	mockCore.On("GetDB").Return(db)
	mockCore.On("RegisterRoute", mock.Anything, mock.Anything, mock.Anything).Return()

	plugin := &WebchatPlugin{}
	err := plugin.OnActivate(mockCore)

	assert.NoError(t, err)
	mockCore.AssertNumberOfCalls(t, "RegisterRoute", 2)
	assert.Len(t, mockCore.registeredRoutes, 2)
	assert.Equal(t, "GET", mockCore.registeredRoutes[0].Method)
	assert.Equal(t, "/webchat/:whatsappId", mockCore.registeredRoutes[0].Path)
	assert.Equal(t, "POST", mockCore.registeredRoutes[1].Method)
	assert.Equal(t, "/webchat/:whatsappId/tickets", mockCore.registeredRoutes[1].Path)
}

// TestWebchatPlugin_GET_ReturnsWhatsAppConfig testa a rota GET com dados reais
func TestWebchatPlugin_GET_ReturnsWhatsAppConfig(t *testing.T) {
	db := setupPluginTestDB(t)
	tenantID := uuid.New()

	whatsapp := models.Whatsapp{
		Name:            "Test Channel",
		GreetingMessage: "Olá!",
		FarewellMessage: "Até logo!",
		TenantID:        tenantID,
	}
	db.Create(&whatsapp)

	mockCore := new(MockWatinkCore)
	mockCore.On("GetDB").Return(db)
	mockCore.On("RegisterRoute", mock.Anything, mock.Anything, mock.Anything).Return()

	plugin := &WebchatPlugin{}
	err := plugin.OnActivate(mockCore)
	assert.NoError(t, err)

	// Simular a rota GET
	r := authedRouter(db, tenantID)
	r.GET("/webchat/:whatsappId", mockCore.registeredRoutes[0].Handler)

	req := httptest.NewRequest("GET", "/webchat/"+strconv.Itoa(whatsapp.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Test Channel", body["name"])
	assert.Equal(t, "Olá!", body["greetingMessage"])
}

// TestWebchatPlugin_GET_NotFound testa 404 quando WhatsApp não existe
func TestWebchatPlugin_GET_NotFound(t *testing.T) {
	db := setupPluginTestDB(t)
	mockCore := new(MockWatinkCore)
	mockCore.On("GetDB").Return(db)
	mockCore.On("RegisterRoute", mock.Anything, mock.Anything, mock.Anything).Return()

	plugin := &WebchatPlugin{}
	if err := plugin.OnActivate(mockCore); err != nil {
		t.Fatal(err)
	}

	r := authedRouter(db, uuid.New())
	r.GET("/webchat/:whatsappId", mockCore.registeredRoutes[0].Handler)

	req := httptest.NewRequest("GET", "/webchat/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestWebchatPlugin_POST_CreatesTicketAndContact testa fluxo completo de POST
func TestWebchatPlugin_POST_CreatesTicketAndContact(t *testing.T) {
	db := setupPluginTestDB(t)
	tenantID := uuid.New()

	whatsapp := models.Whatsapp{
		Name:            "Chat Channel",
		GreetingMessage: "Hi",
		FarewellMessage: "Bye",
		TenantID:        tenantID,
	}
	db.Create(&whatsapp)

	mockCore := new(MockWatinkCore)
	mockCore.On("GetDB").Return(db)
	mockCore.On("RegisterRoute", mock.Anything, mock.Anything, mock.Anything).Return()

	plugin := &WebchatPlugin{}
	if err := plugin.OnActivate(mockCore); err != nil {
		t.Fatal(err)
	}

	r := authedRouter(db, tenantID)
	r.POST("/webchat/:whatsappId/tickets", mockCore.registeredRoutes[1].Handler)

	payload := map[string]string{
		"name":    "João",
		"email":   "joao@test.com",
		"phone":   "5511999990000",
		"message": "Preciso de ajuda",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/webchat/"+strconv.Itoa(whatsapp.ID)+"/tickets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, resp["ticketId"])
	assert.NotNil(t, resp["contactId"])

	// Verificar persistência no DB (MESMA instância, não nova conexão)
	var contact models.Contact
	result := db.Where(`email = ? AND "tenantId" = ?`, "joao@test.com", tenantID).First(&contact)
	assert.NoError(t, result.Error)
	assert.Equal(t, "João", contact.Name)

	var ticket models.Ticket
	result = db.Where(`"contactId" = ?`, contact.ID).First(&ticket)
	assert.NoError(t, result.Error)
	assert.Equal(t, "pending", ticket.Status)
}

// Outra empresa não alcança o webchat nem cria ticket nele: antes o handler buscava o canal só pelo id.
func TestWebchatPlugin_OutraEmpresaNaoAlcancaOCanal(t *testing.T) {
	db := setupPluginTestDB(t)
	dono, intruso := uuid.New(), uuid.New()
	whatsapp := models.Whatsapp{Name: "Canal do dono", TenantID: dono}
	db.Create(&whatsapp)

	mockCore := new(MockWatinkCore)
	mockCore.On("GetDB").Return(db)
	mockCore.On("RegisterRoute", mock.Anything, mock.Anything, mock.Anything).Return()
	assert.NoError(t, (&WebchatPlugin{}).OnActivate(mockCore))

	r := authedRouter(db, intruso)
	r.GET("/webchat/:whatsappId", mockCore.registeredRoutes[0].Handler)
	r.POST("/webchat/:whatsappId/tickets", mockCore.registeredRoutes[1].Handler)

	id := strconv.Itoa(whatsapp.ID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/webchat/"+id, nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	req := httptest.NewRequest("POST", "/webchat/"+id+"/tickets", bytes.NewReader([]byte(`{"name":"X","email":"x@x.io","phone":"1"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var n int64
	db.Model(&models.Ticket{}).Count(&n)
	assert.Zero(t, n, "nenhum ticket pode ser criado")
}

// ---- OnDeactivate smoke tests ----

func TestWebchatPlugin_OnDeactivate(t *testing.T) {
	mockCore := new(MockWatinkCore)
	p := &WebchatPlugin{}
	assert.NoError(t, p.OnDeactivate(mockCore))
}

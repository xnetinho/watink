package controllers

import (
	"net/http"
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

func phoneRouter(db *gorm.DB, tenant uuid.UUID, pub *mockPublisher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	ctrl := NewContactController(repository.NewGORMContactRepo(db), nil, pub, nil)
	r := gin.New()
	r.Use(testScopedMiddleware(db, tenant.String()))
	r.POST("/contacts", ctrl.CreateContact)
	return r
}

func createdContact(t *testing.T, db *gorm.DB, tenant uuid.UUID, number string) models.Contact {
	t.Helper()
	var c models.Contact
	require.NoError(t, db.Where(`"tenantId" = ? AND number = ?`, tenant, number).First(&c).Error)
	return c
}

func TestCreateContact_WhatsappConectadoMandaGravarNaAgendaDoCelular(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	pub := &mockPublisher{}
	wa := models.Whatsapp{Name: "Comercial", Status: "CONNECTED", TenantID: tenant}
	require.NoError(t, db.Create(&wa).Error)

	w := doJSON(phoneRouter(db, tenant, pub), http.MethodPost, "/contacts", map[string]interface{}{"name": "Maria", "number": "5583999990001", "connectionId": wa.ID})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.Equal(t, "wbot."+tenant.String()+"."+strconv.Itoa(wa.ID)+".contact.save", pub.routingKey, "o comando vai para a conexão escolhida")
	assert.Equal(t, "pending", createdContact(t, db, tenant, "5583999990001").PhoneSync)
}

func TestCreateContact_WhatsappDesconectadoCadastraMasAvisaQueNaoFoiAoCelular(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	pub := &mockPublisher{}
	wa := models.Whatsapp{Name: "Comercial", Status: "DISCONNECTED", TenantID: tenant}
	require.NoError(t, db.Create(&wa).Error)

	w := doJSON(phoneRouter(db, tenant, pub), http.MethodPost, "/contacts", map[string]interface{}{"name": "Maria", "number": "5583999990002", "connectionId": wa.ID})
	require.Equal(t, http.StatusOK, w.Code, "o cadastro não depende do celular")

	c := createdContact(t, db, tenant, "5583999990002")
	assert.Equal(t, "failed", c.PhoneSync)
	assert.Equal(t, "connection_not_connected", c.PhoneSyncError)
	assert.False(t, pub.called, "sem sessão não há para onde enviar")
}

func TestCreateContact_TelegramNaoTemAgendaDeCelular(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	pub := &mockPublisher{}
	ch := models.ChannelConnection{TenantID: tenant, Kind: "telegram", Name: "Jarvis", Status: "CONNECTED"}
	require.NoError(t, db.Create(&ch).Error)

	w := doJSON(phoneRouter(db, tenant, pub), http.MethodPost, "/contacts", map[string]interface{}{"name": "Ana", "number": "tg-ana", "connectionSource": "channel", "connectionId": ch.ID})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	c := createdContact(t, db, tenant, "tg-ana")
	assert.Equal(t, "channel", c.ConnectionSource)
	assert.Equal(t, ch.ID, c.ConnectionID)
	assert.Empty(t, c.PhoneSync, "só o WhatsApp tem agenda no celular")
	assert.False(t, pub.called)
}

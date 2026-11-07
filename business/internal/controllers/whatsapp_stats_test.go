package controllers

import (
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
)

// As estatísticas da conexão contam os tickets e as mensagens do dia dela, e só dela. O endpoint não tinha teste e a coluna
// antiga (whatsappId) só falhou na tela.
func TestStatsWhatsapp_ContaSoOsDaConexao(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T Stats", Status: "active"}).Error)

	a := models.WhatsAppView{Name: "A", TenantID: tenant, Status: "CONNECTED"}
	b := models.WhatsAppView{Name: "B", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, models.CreateWhatsapp(db, &a))
	require.NoError(t, models.CreateWhatsapp(db, &b))
	contact := models.Contact{Name: "C", Number: "5511999990000", TenantID: tenant}
	require.NoError(t, db.Create(&contact).Error)

	mk := func(conn int, key string, msgs int) {
		tk := models.Ticket{TenantID: tenant, ContactID: contact.ID, ChannelConnectionID: conn, ConversationKey: key, Status: "open"}
		require.NoError(t, db.Create(&tk).Error)
		for i := 0; i < msgs; i++ {
			require.NoError(t, db.Create(&models.Message{ID: key + strconv.Itoa(i), TicketID: tk.ID, TenantID: tenant, Body: "x", ContactID: &contact.ID}).Error)
		}
	}
	mk(a.ID, "wa:a1", 2)
	mk(a.ID, "wa:a2", 1)
	mk(b.ID, "wa:b1", 5)

	wc := NewWhatsappController(repository.NewGORMChannelSessionRepo(db), nil, nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenantId", tenant.String()); c.Set("db", db); c.Next() })
	r.GET("/whatsapp/:id/stats", wc.StatsWhatsapp)

	req := httptest.NewRequest(http.MethodGet, "/whatsapp/"+strconv.Itoa(a.ID)+"/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.EqualValues(t, 2, out["tickets"], w.Body.String())
	assert.EqualValues(t, 3, out["messagesToday"], w.Body.String())
}

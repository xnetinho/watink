package controllers

import (
	"bytes"
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

func putPolicy(t *testing.T, r *gin.Engine, id int, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/whatsapp/"+strconv.Itoa(id)+"/presence-policy", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdatePresencePolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T Presença", Status: "active"}).Error)
	wa := models.Whatsapp{Name: "w", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	require.Equal(t, "never", wa.PresencePolicy, "padrão do modelo: não silencia o celular")

	wc := NewWhatsappController(repository.NewGORMChannelSessionRepo(db), nil, nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenantId", tenant.String()); c.Set("db", db); c.Next() })
	r.PUT("/whatsapp/:id/presence-policy", wc.UpdatePresencePolicy)

	read := func() models.Whatsapp {
		var w models.Whatsapp
		require.NoError(t, db.First(&w, wa.ID).Error)
		return w
	}

	t.Run("on_demand com tempo", func(t *testing.T) {
		w := putPolicy(t, r, wa.ID, `{"presencePolicy":"on_demand","presenceIdleMinutes":15}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		got := read()
		assert.Equal(t, "on_demand", got.PresencePolicy)
		assert.Equal(t, 15, got.PresenceIdleMinutes)
	})
	t.Run("mudar a política sem mandar o tempo preserva o tempo", func(t *testing.T) {
		require.Equal(t, http.StatusOK, putPolicy(t, r, wa.ID, `{"presencePolicy":"always"}`).Code)
		got := read()
		assert.Equal(t, "always", got.PresencePolicy)
		assert.Equal(t, 15, got.PresenceIdleMinutes)
	})
	for name, body := range map[string]string{
		"política inventada":  `{"presencePolicy":"sempre"}`,
		"vazia":               `{"presencePolicy":""}`,
		"maiúscula":           `{"presencePolicy":"ALWAYS"}`,
		"tempo zero":          `{"presencePolicy":"on_demand","presenceIdleMinutes":0}`,
		"tempo acima do teto": `{"presencePolicy":"on_demand","presenceIdleMinutes":241}`,
		"tempo negativo":      `{"presencePolicy":"on_demand","presenceIdleMinutes":-1}`,
	} {
		t.Run("recusa: "+name, func(t *testing.T) {
			before := read()
			assert.Equal(t, http.StatusBadRequest, putPolicy(t, r, wa.ID, body).Code)
			assert.Equal(t, before.PresencePolicy, read().PresencePolicy, "recusado não pode alterar nada")
		})
	}
	t.Run("conexão de outra empresa é 404", func(t *testing.T) {
		other := models.Whatsapp{Name: "alheia", TenantID: uuid.New(), Status: "CONNECTED"}
		require.NoError(t, db.Create(&models.Tenant{ID: other.TenantID, Name: "Outra", Status: "active"}).Error)
		require.NoError(t, db.Create(&other).Error)
		assert.Equal(t, http.StatusNotFound, putPolicy(t, r, other.ID, `{"presencePolicy":"always"}`).Code)
		var got models.Whatsapp
		require.NoError(t, db.First(&got, other.ID).Error)
		assert.Equal(t, "never", got.PresencePolicy)
	})
}

package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A lista única devolve as conexões de todos os canais da empresa, com connectionId e sem a chave antiga whatsappId, e não
// mistura as de outra empresa.
func TestConnectionRouter_List_TodosOsCanaisDaEmpresa(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant, other := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T", Status: "active"}).Error)
	require.NoError(t, db.Create(&models.Tenant{ID: other, Name: "O", Status: "active"}).Error)

	wa := models.WhatsAppView{Name: "Zap", TenantID: tenant, Status: "CONNECTED", Number: "5511999990000"}
	require.NoError(t, models.CreateWhatsapp(db, &wa))
	tg := models.ChannelConnection{Kind: "telegram", Name: "Bot", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&tg).Error)
	alheia := models.WhatsAppView{Name: "Alheia", TenantID: other, Status: "CONNECTED"}
	require.NoError(t, models.CreateWhatsapp(db, &alheia))

	router := NewConnectionRouter(NewWhatsappController(repository.NewGORMChannelSessionRepo(db), nil, nil, nil), nil, NewChannelConnectionController())
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenantId", tenant.String())
		c.Set("db", db)
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.GET("/channels/connections", router.List)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/channels/connections", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var out []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Len(t, out, 2)
	kinds := map[string]bool{}
	for _, c := range out {
		kinds[c["kind"].(string)] = true
		assert.NotContains(t, c, "whatsappId")
		assert.EqualValues(t, c["id"], c["connectionId"])
	}
	assert.True(t, kinds["whatsapp"] && kinds["telegram"], "%v", kinds)
}

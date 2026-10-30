package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/controllers"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (e *sendEnv) receiptsRouter(tenant uuid.UUID) *gin.Engine {
	mc := controllers.NewMessageController(nil, nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", e.db)
		c.Set("tenantId", tenant.String())
		c.Set("userId", float64(e.userID))
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.GET("/message/:messageId/receipts", mc.ListMessageReceipts)
	return r
}

func TestListMessageReceipts(t *testing.T) {
	e := newSendEnv(t)
	tk := e.newTicket(t, "chat-R")
	require.NoError(t, e.db.Create(&models.Message{ID: "R1", Body: "Oi", FromMe: true, TicketID: tk.ID, TenantID: e.tenantID}).Error)
	require.NoError(t, e.db.Create(&models.Message{ID: "SEM_RECIBO", Body: "x", FromMe: true, TicketID: tk.ID, TenantID: e.tenantID}).Error)
	at := time.Date(2026, 10, 30, 14, 0, 0, 0, time.UTC)
	for _, rc := range []models.MessageReceipt{
		{TenantID: e.tenantID, MessageID: "R1", Stage: models.ReceiptDelivered, At: at},
		{TenantID: e.tenantID, MessageID: "R1", Stage: models.ReceiptRead, At: at.Add(5 * time.Minute)},
	} {
		require.NoError(t, e.db.Create(&rc).Error)
	}
	get := func(r *gin.Engine, id string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("GET", "/message/"+id+"/receipts", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("devolve entrega e leitura com o horário", func(t *testing.T) {
		w := get(e.receiptsRouter(e.tenantID), "R1")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var out struct {
			Receipts []struct {
				Delivered *string `json:"delivered"`
				Read      *string `json:"read"`
				Played    *string `json:"played"`
			} `json:"receipts"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		require.Len(t, out.Receipts, 1)
		assert.Equal(t, "2026-10-30T14:00:00Z", *out.Receipts[0].Delivered)
		assert.Equal(t, "2026-10-30T14:05:00Z", *out.Receipts[0].Read)
		assert.Nil(t, out.Receipts[0].Played)
		assert.NotContains(t, w.Body.String(), "tenantId", "o id da empresa não sai na API")
	})
	t.Run("mensagem sem recibos: lista vazia (não é erro)", func(t *testing.T) {
		w := get(e.receiptsRouter(e.tenantID), "SEM_RECIBO")
		require.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"receipts":[]}`, w.Body.String())
	})
	t.Run("outra empresa recebe 404", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get(e.receiptsRouter(uuid.New()), "R1").Code)
	})
	t.Run("mensagem inexistente: 404", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get(e.receiptsRouter(e.tenantID), "NAO_EXISTE").Code)
	})
}

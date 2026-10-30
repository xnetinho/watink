package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/controllers"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (e *sendEnv) editsRouter(tenant uuid.UUID) *gin.Engine {
	mc := controllers.NewMessageController(nil, nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", e.db)
		c.Set("tenantId", tenant.String())
		c.Set("userId", float64(e.userID))
		c.Set("alcance", "tenant")
		c.Next()
	})
	r.GET("/message/:messageId/edits", mc.ListMessageEdits)
	return r
}

func TestListMessageEdits(t *testing.T) {
	e := newSendEnv(t)
	tk := e.newTicket(t, "chat-E")
	t0 := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	require.NoError(t, e.db.Create(&models.Message{ID: "ED1", Body: "Oi", TicketID: tk.ID, TenantID: e.tenantID, CreatedAt: t0}).Error)
	require.NoError(t, e.db.Create(&models.Message{ID: "SEM_EDICAO", Body: "x", TicketID: tk.ID, TenantID: e.tenantID, CreatedAt: t0}).Error)
	for i, b := range []string{"Olá", "Olá, tudo bem?"} {
		_, err := services.RecordEdit(t.Context(), e.db, services.RecordEditInput{TenantID: e.tenantID, MessageID: "ED1", Body: b, EditedAt: t0.Add(time.Duration(i+1) * time.Minute)})
		require.NoError(t, err)
	}
	get := func(r *gin.Engine, id string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("GET", "/message/"+id+"/edits", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("devolve a cadeia em ordem, da original à mais recente", func(t *testing.T) {
		w := get(e.editsRouter(e.tenantID), "ED1")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var out struct {
			Edits []struct {
				Version  int    `json:"version"`
				Body     string `json:"body"`
				EditedBy string `json:"editedBy"`
			} `json:"edits"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		require.Len(t, out.Edits, 3)
		assert.Equal(t, []int{1, 2, 3}, []int{out.Edits[0].Version, out.Edits[1].Version, out.Edits[2].Version})
		assert.Equal(t, "Oi", out.Edits[0].Body)
		assert.Equal(t, "Olá, tudo bem?", out.Edits[2].Body)
		assert.NotContains(t, w.Body.String(), "tenantId", "o id da empresa não sai na API")
	})

	t.Run("mensagem nunca editada: lista vazia (não é erro)", func(t *testing.T) {
		w := get(e.editsRouter(e.tenantID), "SEM_EDICAO")
		require.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"edits":[]}`, w.Body.String())
	})

	t.Run("outra empresa recebe 404, nunca a cadeia", func(t *testing.T) {
		w := get(e.editsRouter(uuid.New()), "ED1")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NotContains(t, w.Body.String(), "Olá")
	})
	t.Run("mensagem inexistente: 404", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get(e.editsRouter(e.tenantID), "NAO_EXISTE").Code)
	})
}

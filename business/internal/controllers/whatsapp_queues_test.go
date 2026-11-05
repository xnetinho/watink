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
	"gorm.io/gorm"
)

type noLimit struct{}

func (noLimit) CheckLimit(uuid.UUID, string) error { return nil }

func queuesOf(t *testing.T, db *gorm.DB, connID int) []int {
	t.Helper()
	var ids []int
	require.NoError(t, db.Raw(`SELECT queue_id FROM channel_connection_queues WHERE channel_connection_id = ? ORDER BY queue_id`, connID).Scan(&ids).Error)
	return ids
}

func TestWhatsapp_QueueIDs_CriaEditaEIsola(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	tenant, other := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{tenant, other} {
		require.NoError(t, db.Create(&models.Tenant{ID: id, Name: "T " + id.String(), Status: "active"}).Error)
	}
	q1 := models.Queue{Name: "F1", Color: "#111", TenantID: tenant}
	q2 := models.Queue{Name: "F2", Color: "#222", TenantID: tenant}
	qOther := models.Queue{Name: "FX", Color: "#333", TenantID: other}
	require.NoError(t, db.Create(&q1).Error)
	require.NoError(t, db.Create(&q2).Error)
	require.NoError(t, db.Create(&qOther).Error)

	wc := NewWhatsappController(repository.NewGORMChannelSessionRepo(db), noLimit{}, nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenantId", tenant.String()); c.Set("db", db); c.Next() })
	r.POST("/whatsapp", wc.CreateWhatsapp)
	r.PUT("/whatsapp/:id", wc.UpdateWhatsapp)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	w := call(http.MethodPost, "/whatsapp", `{"name":"zap","queueIds":[`+strconv.Itoa(q1.ID)+`,`+strconv.Itoa(qOther.ID)+`]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created models.Whatsapp
	require.NoError(t, models.FirstWhatsapp(db.Where("name = ?", "zap"), &created))
	assert.Equal(t, []int{q1.ID}, queuesOf(t, db, created.ID), "fila de outra empresa é ignorada")

	url := "/whatsapp/" + strconv.Itoa(created.ID)
	assert.Equal(t, http.StatusOK, call(http.MethodPut, url, `{"name":"zap2"}`).Code)
	assert.Equal(t, []int{q1.ID}, queuesOf(t, db, created.ID), "PUT sem queueIds não mexe nas filas")

	assert.Equal(t, http.StatusOK, call(http.MethodPut, url, `{"queueIds":[`+strconv.Itoa(q1.ID)+`,`+strconv.Itoa(q2.ID)+`]}`).Code)
	assert.Equal(t, []int{q1.ID, q2.ID}, queuesOf(t, db, created.ID))

	assert.Equal(t, http.StatusOK, call(http.MethodPut, url, `{"queueIds":[]}`).Code)
	assert.Empty(t, queuesOf(t, db, created.ID), "queueIds vazio desliga todas")
}

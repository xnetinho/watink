package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alltomatos/watinkdev/business/internal/models"
)

// O contrato que a tela lê: 409 com os números do que depende da fila, 422 para destino inválido, 200 quando exclui.
func TestDeleteQueue_ContratoHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupQueueTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T"}).Error)
	from := models.Queue{Name: "origem", Color: "#000", TenantID: tenant}
	to := models.Queue{Name: "destino", Color: "#000", TenantID: tenant}
	require.NoError(t, db.Create(&from).Error)
	require.NoError(t, db.Create(&to).Error)
	require.NoError(t, db.Create(&models.Ticket{Status: "pending", TenantID: tenant, QueueID: &from.ID, ChannelConnectionID: 1}).Error)

	call := func(query string) (int, map[string]any) {
		path := fmt.Sprintf("/queue/%d%s", from.ID, query)
		c, w := setupQueueContext(t, db, tenant, "DELETE", path, nil)
		c.Params = gin.Params{{Key: "queueId", Value: fmt.Sprintf("%d", from.ID)}}
		NewQueueController().DeleteQueue(c)
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return w.Code, body
	}

	code, body := call("")
	assert.Equal(t, http.StatusConflict, code)
	assert.Equal(t, "ERR_QUEUE_HAS_DEPENDENTS", body["error"])
	assert.EqualValues(t, 1, body["dependents"].(map[string]any)["Tickets"])

	code, body = call("?moveTo=" + fmt.Sprint(from.ID))
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.Equal(t, "ERR_QUEUE_MOVE_INVALID", body["error"])

	code, _ = call("?moveTo=abc")
	assert.Equal(t, http.StatusBadRequest, code)

	code, _ = call("?moveTo=" + fmt.Sprint(to.ID))
	assert.Equal(t, http.StatusOK, code)
}

func TestDeleteUser_ContratoHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T"}).Error)
	u := models.User{Name: "u", Email: "u@x", PasswordHash: "h", TenantID: tenant}
	dest := models.User{Name: "d", Email: "d@x", PasswordHash: "h", TenantID: tenant}
	require.NoError(t, db.Create(&u).Error)
	require.NoError(t, db.Create(&dest).Error)
	require.NoError(t, db.Create(&models.Contact{Name: "c", Number: "1", TenantID: tenant, WalletUserID: &u.ID}).Error)

	call := func(query string) (int, map[string]any) {
		c, w := setupUserContext(t, db, tenant, "DELETE", fmt.Sprintf("/users/%d%s", u.ID, query), nil)
		c.Params = gin.Params{{Key: "userId", Value: fmt.Sprintf("%d", u.ID)}}
		NewUserController(&mockUserRepo{db: db}, &mockPlanLimit{}).DeleteUser(c)
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return w.Code, body
	}

	code, body := call("")
	assert.Equal(t, http.StatusConflict, code)
	assert.Equal(t, "ERR_USER_HAS_WALLET", body["error"])
	assert.EqualValues(t, 1, body["contacts"])

	code, body = call("?walletTo=" + fmt.Sprint(u.ID))
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.Equal(t, "ERR_WALLET_TARGET_INVALID", body["error"])

	code, _ = call("?walletTo=" + fmt.Sprint(dest.ID))
	assert.Equal(t, http.StatusOK, code)
}

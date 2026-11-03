package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func waVersionCall(t *testing.T, ctrl *WAVersionController, method, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(method, "/system/wa-version", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	if method == http.MethodGet {
		ctrl.Get(c)
	} else {
		ctrl.Save(c)
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestWAVersion_SalvaMesmoComEngineForaDoAr(t *testing.T) {
	t.Setenv("ENGINE_HOST", "")
	t.Setenv("GROUPS_API_TOKEN", "")
	ctrl := NewWAVersionController(services.NewWAVersionService(testutil.NewTestDB(t)))

	w, out := waVersionCall(t, ctrl, http.MethodPut, `{"mode":"manual","manualVersion":"2.3000.1052747707"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "manual", out["mode"])
	assert.Equal(t, false, out["engineApplied"], "a tela precisa saber que o engine ainda não aplicou")

	w, out = waVersionCall(t, ctrl, http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "manual", out["mode"])
	assert.Equal(t, "2.3000.1052747707", out["manualVersion"])
	assert.Equal(t, "engine_unavailable", out["engineError"])
}

func TestWAVersion_RecusaVersaoInvalida(t *testing.T) {
	ctrl := NewWAVersionController(services.NewWAVersionService(testutil.NewTestDB(t)))
	w, out := waVersionCall(t, ctrl, http.MethodPut, `{"mode":"manual","manualVersion":"lixo"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid_wa_version", out["error"])

	w, out = waVersionCall(t, ctrl, http.MethodPut, `{"mode":"outro"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid_wa_version_mode", out["error"])

	w, _ = waVersionCall(t, ctrl, http.MethodPut, `não é json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

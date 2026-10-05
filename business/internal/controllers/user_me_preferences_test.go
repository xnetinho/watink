package controllers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMyTheme_SavesAndMergesWithoutLosingOtherConfigs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	db.Exec(`INSERT INTO "Users" (name, email, "passwordHash", "tenantId", alcance, configs) VALUES (?,?,?,?,?,?)`,
		"Pessoa", "p@test.com", "hash", tenantID, "proprio", `{"dashboard":{"widgets":[{"id":"a","visible":true}]}}`)
	var uid int
	db.Raw(`SELECT LASTVAL()`).Scan(&uid)
	// Repositório GORM real: o mockUserRepo.Update só persiste name/email/alcance
	// e ignoraria configs, escondendo exatamente o que este teste quer provar.
	ctrl := NewUserController(repository.NewGORMUserRepo(db), &mockPlanLimit{})

	call := func(theme string) (int, string) {
		payload, _ := json.Marshal(map[string]interface{}{"theme": theme})
		c, w := setupUserContext(t, db, tenantID, "PUT", "/me/theme", payload)
		c.Set("alcance", "proprio")
		c.Set("userId", float64(uid))
		ctrl.UpdateMyTheme(c)
		return w.Code, w.Body.String()
	}
	read := func() map[string]interface{} {
		var raw string
		db.Raw(`SELECT configs::text FROM "Users" WHERE id = ?`, uid).Scan(&raw)
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(raw), &m), raw)
		return m
	}

	code, body := call("dark")
	require.Equal(t, http.StatusOK, code, body)
	cfg := read()
	assert.Equal(t, "dark", cfg["theme"])
	assert.NotNil(t, cfg["dashboard"], "o merge não pode apagar as preferências do dashboard")

	code, _ = call("light")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "light", read()["theme"], "segunda troca sobrescreve só o tema")
	assert.NotNil(t, read()["dashboard"])
}

func TestUpdateMyTheme_RejectsInvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	db.Exec(`INSERT INTO "Users" (name, email, "passwordHash", "tenantId", alcance, configs) VALUES (?,?,?,?,?,?)`,
		"Pessoa", "q@test.com", "hash", tenantID, "proprio", `{}`)
	var uid int
	db.Raw(`SELECT LASTVAL()`).Scan(&uid)
	ctrl := NewUserController(&mockUserRepo{db: db}, &mockPlanLimit{})

	for _, bad := range []string{"", "auto", "DARK", "<script>", "blue"} {
		payload, _ := json.Marshal(map[string]interface{}{"theme": bad})
		c, w := setupUserContext(t, db, tenantID, "PUT", "/me/theme", payload)
		c.Set("userId", float64(uid))
		ctrl.UpdateMyTheme(c)
		assert.Equal(t, http.StatusBadRequest, w.Code, "valor %q deveria ser rejeitado", bad)
	}
}

func TestUpdateMyTheme_OnlyTouchesTheAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantID := uuid.New()
	for _, e := range []string{"a@test.com", "b@test.com"} {
		db.Exec(`INSERT INTO "Users" (name, email, "passwordHash", "tenantId", alcance, configs) VALUES (?,?,?,?,?,?)`,
			e, e, "hash", tenantID, "proprio", `{}`)
	}
	var idA, idB int
	db.Raw(`SELECT id FROM "Users" WHERE email = 'a@test.com'`).Scan(&idA)
	db.Raw(`SELECT id FROM "Users" WHERE email = 'b@test.com'`).Scan(&idB)
	ctrl := NewUserController(&mockUserRepo{db: db}, &mockPlanLimit{})

	payload, _ := json.Marshal(map[string]interface{}{"theme": "dark"})
	c, w := setupUserContext(t, db, tenantID, "PUT", "/me/theme", payload)
	c.Set("userId", float64(idA))
	ctrl.UpdateMyTheme(c)
	require.Equal(t, http.StatusOK, w.Code)

	var cfgB string
	db.Raw(`SELECT configs::text FROM "Users" WHERE id = ?`, idB).Scan(&cfgB)
	assert.NotContains(t, cfgB, "dark", "o tema de outro usuário não pode mudar")
}

func TestUpdateMyTheme_OtherTenantUserIsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupUserTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	db.Exec(`INSERT INTO "Users" (name, email, "passwordHash", "tenantId", alcance, configs) VALUES (?,?,?,?,?,?)`,
		"B", "tb@test.com", "hash", tenantB, "proprio", `{}`)
	var idB int
	db.Raw(`SELECT LASTVAL()`).Scan(&idB)
	ctrl := NewUserController(&mockUserRepo{db: db}, &mockPlanLimit{})

	payload, _ := json.Marshal(map[string]interface{}{"theme": "dark"})
	c, w := setupUserContext(t, db, tenantA, "PUT", "/me/theme", payload)
	c.Set("userId", float64(idB))
	ctrl.UpdateMyTheme(c)
	assert.Equal(t, http.StatusNotFound, w.Code, "usuário de outro tenant não pode ser alterado")
}

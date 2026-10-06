package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/calls"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type callPub struct {
	mu   sync.Mutex
	cmds []string
}

func (p *callPub) PublishCommand(key string, payload interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cmds = append(p.cmds, key)
	return nil
}

// sent conta os comandos publicados cuja routing key termina em suffix.
func (p *callPub) sent(suffix string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, c := range p.cmds {
		if len(c) >= len(suffix) && c[len(c)-len(suffix):] == suffix {
			n++
		}
	}
	return n
}

type callWorld struct {
	db     *gorm.DB
	tenant uuid.UUID
	other  uuid.UUID
	pub    *callPub
	svc    *calls.Service
	wa     models.Whatsapp
	ticket models.Ticket
	cargos map[string]int
}

func newCallWorld(t *testing.T) *callWorld {
	t.Helper()
	db := testutil.NewTestDB(t)
	w := &callWorld{db: db, tenant: uuid.New(), other: uuid.New(), pub: &callPub{}, cargos: map[string]int{}}
	w.svc = calls.NewService(db, repository.NewGORMContactRepo(db), repository.NewGORMTicketRepo(db),
		repository.NewGORMQueueRepo(db), w.pub, nil, nil)
	w.wa = models.Whatsapp{Name: "WA" + w.tenant.String(), TenantID: w.tenant, Status: "CONNECTED", ProxyMode: "none"}
	require.NoError(t, db.Create(&w.wa).Error)
	contact := models.Contact{Name: "Cliente", Number: "5511999991111", TenantID: w.tenant}
	require.NoError(t, db.Create(&contact).Error)
	w.ticket = models.Ticket{Status: "open", ContactID: contact.ID, WhatsappID: w.wa.ID, TenantID: w.tenant}
	require.NoError(t, db.Create(&w.ticket).Error)
	for _, a := range []string{"receive", "place", "read", "delete", "manage"} {
		seedPermission(t, db, "calls", a)
	}
	return w
}

func (w *callWorld) user(t *testing.T, name, alcance string, tenant uuid.UUID, perms ...string) models.User {
	t.Helper()
	cargoID := seedCargo(t, w.db, tenant, "cargo-"+name)
	for _, p := range perms {
		var id int
		require.NoError(t, w.db.Raw(`SELECT id FROM "Permissions" WHERE resource = 'calls' AND action = ?`, p).Scan(&id).Error)
		require.NoError(t, w.db.Create(&models.CargoPermissao{CargoID: cargoID, PermissionID: id}).Error)
	}
	u := models.User{Name: name, Email: name + "@" + tenant.String()[:6] + ".io", TenantID: tenant, CargoID: &cargoID, Alcance: alcance}
	require.NoError(t, w.db.Create(&u).Error)
	return u
}

func (w *callWorld) router(u models.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", w.db)
		c.Set("tenantId", u.TenantID)
		c.Set("alcance", u.Alcance)
		c.Set("userId", float64(u.ID))
		c.Next()
	})
	cc := NewCallController(w.svc)
	r.PUT("/calls/pause", auth.RequirePermission("calls", "receive"), cc.Pause)
	r.POST("/calls", auth.RequirePermission("calls", "place"), cc.Place)
	r.GET("/calls", auth.RequirePermission("calls", "read"), cc.List)
	r.GET("/calls/:id", auth.RequirePermission("calls", "read"), cc.Show)
	r.POST("/calls/:id/accept", auth.RequirePermission("calls", "receive"), cc.Accept)
	r.POST("/calls/:id/reject", auth.RequirePermission("calls", "receive"), cc.Reject)
	r.POST("/calls/:id/end", auth.RequireAnyPermission([2]string{"calls", "receive"}, [2]string{"calls", "place"}), cc.End)
	return r
}

func do(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func code(w *httptest.ResponseRecorder) string {
	var m map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	s, _ := m["code"].(string)
	return s
}

// 7.1: cada rota exige a permissão certa.
func TestCallRoutes_PermissionMatrix(t *testing.T) {
	w := newCallWorld(t)
	none := w.user(t, "nada", "proprio", w.tenant)
	rN := w.router(none)
	for _, c := range []struct{ method, path string }{
		{"POST", "/calls"}, {"GET", "/calls"}, {"GET", "/calls/X"}, {"POST", "/calls/X/accept"},
		{"POST", "/calls/X/reject"}, {"POST", "/calls/X/end"}, {"PUT", "/calls/pause"},
	} {
		assert.Equal(t, http.StatusForbidden, do(rN, c.method, c.path, map[string]interface{}{"ticketId": 1, "paused": true}).Code, "%s %s sem permissão", c.method, c.path)
	}

	onlyRead := w.user(t, "leitor", "proprio", w.tenant, "read")
	assert.Equal(t, http.StatusOK, do(w.router(onlyRead), "GET", "/calls", nil).Code)
	assert.Equal(t, http.StatusForbidden, do(w.router(onlyRead), "POST", "/calls", map[string]interface{}{"ticketId": w.ticket.ID}).Code)
}

func TestCallEnd_AcceptsReceiveOrPlace(t *testing.T) {
	w := newCallWorld(t)
	rcv := w.user(t, "rcv", "proprio", w.tenant, "receive")
	plc := w.user(t, "plc", "proprio", w.tenant, "place")
	rd := w.user(t, "rd", "proprio", w.tenant, "read")
	assert.NotEqual(t, http.StatusForbidden, do(w.router(rcv), "POST", "/calls/X/end", nil).Code)
	assert.NotEqual(t, http.StatusForbidden, do(w.router(plc), "POST", "/calls/X/end", nil).Code, "quem liga também precisa poder desligar")
	assert.Equal(t, http.StatusForbidden, do(w.router(rd), "POST", "/calls/X/end", nil).Code)
}

// 7.2: POST /calls e cada negativa.
func TestPlaceCall_Validations(t *testing.T) {
	w := newCallWorld(t)
	u := w.user(t, "ligador", "proprio", w.tenant, "place")
	require.NoError(t, w.db.Model(&models.Ticket{}).Where("id = ?", w.ticket.ID).Update("userId", u.ID).Error)
	r := w.router(u)
	body := map[string]interface{}{"ticketId": w.ticket.ID}

	res := do(r, "POST", "/calls", body)
	require.Equal(t, http.StatusAccepted, res.Code, res.Body.String())
	assert.Equal(t, 1, w.pub.sent(".call.start"), "o comando de ligar vai ao engine")
	var l models.CallLog
	require.NoError(t, w.db.Where(`"tenantId" = ?`, w.tenant).First(&l).Error)
	assert.Equal(t, "outgoing", l.Direction)
	require.NotNil(t, l.HandledByUserID)
	assert.Equal(t, u.ID, *l.HandledByUserID)

	res = do(r, "POST", "/calls", body)
	assert.Equal(t, http.StatusConflict, res.Code, "operador já em chamada")
	assert.Equal(t, "USER_BUSY", code(res))
	assert.Equal(t, 1, w.pub.sent(".call.start"), "nenhum segundo comando")
}

func TestPlaceCall_RejectedCases(t *testing.T) {
	w := newCallWorld(t)
	u := w.user(t, "ligador", "tenant", w.tenant, "place")
	r := w.router(u)

	assert.Equal(t, http.StatusBadRequest, do(r, "POST", "/calls", map[string]interface{}{}).Code, "sem ticketId")
	assert.Equal(t, http.StatusNotFound, do(r, "POST", "/calls", map[string]interface{}{"ticketId": 999999}).Code, "ticket inexistente")

	require.NoError(t, w.db.Model(&models.Ticket{}).Where("id = ?", w.ticket.ID).Update("isGroup", true).Error)
	res := do(r, "POST", "/calls", map[string]interface{}{"ticketId": w.ticket.ID})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
	assert.Equal(t, "NOT_INDIVIDUAL", code(res), "grupo/comunidade/canal não liga")
	require.NoError(t, w.db.Model(&models.Ticket{}).Where("id = ?", w.ticket.ID).Update("isGroup", false).Error)

	require.NoError(t, w.db.Model(&models.Whatsapp{}).Where("id = ?", w.wa.ID).Update("status", "DISCONNECTED").Error)
	res = do(r, "POST", "/calls", map[string]interface{}{"ticketId": w.ticket.ID})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
	assert.Equal(t, "NOT_CONNECTED", code(res))
	require.NoError(t, w.db.Model(&models.Whatsapp{}).Where("id = ?", w.wa.ID).Update("status", "CONNECTED").Error)

	require.NoError(t, w.db.Model(&models.Whatsapp{}).Where("id = ?", w.wa.ID).Update("proxyMode", "single").Error)
	res = do(r, "POST", "/calls", map[string]interface{}{"ticketId": w.ticket.ID})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
	assert.Equal(t, "PROXY_BLOCKED", code(res), "conexão com proxy nunca liga")

	assert.Zero(t, w.pub.sent(".call.start"), "nenhuma negativa pode chegar ao engine")
}

func TestPlaceCall_ProxyViaProxyIDAndGroupAlsoBlocks(t *testing.T) {
	w := newCallWorld(t)
	r := w.router(w.user(t, "ligador", "tenant", w.tenant, "place"))
	require.NoError(t, w.db.Model(&models.Whatsapp{}).Where("id = ?", w.wa.ID).Updates(map[string]interface{}{"proxyMode": "none", "proxyGroupId": 3}).Error)
	res := do(r, "POST", "/calls", map[string]interface{}{"ticketId": w.ticket.ID})
	assert.Equal(t, "PROXY_BLOCKED", code(res), "grupo de proxy configurado (mesmo sem pick atual) bloqueia: fail-closed")
}

func TestPlaceCall_OtherTenantsTicketIs404(t *testing.T) {
	w := newCallWorld(t)
	intruder := w.user(t, "intruso", "tenant", w.other, "place")
	res := do(w.router(intruder), "POST", "/calls", map[string]interface{}{"ticketId": w.ticket.ID})
	assert.Equal(t, http.StatusNotFound, res.Code, "ticket de outra empresa não existe para quem liga")
	assert.Zero(t, w.pub.sent(".call.start"))
}

func TestPlaceCall_ConnectionBusy(t *testing.T) {
	w := newCallWorld(t)
	a := w.user(t, "a", "tenant", w.tenant, "place")
	b := w.user(t, "b", "tenant", w.tenant, "place")
	require.Equal(t, http.StatusAccepted, do(w.router(a), "POST", "/calls", map[string]interface{}{"ticketId": w.ticket.ID}).Code)
	res := do(w.router(b), "POST", "/calls", map[string]interface{}{"ticketId": w.ticket.ID})
	assert.Equal(t, http.StatusConflict, res.Code)
	assert.Equal(t, 1, w.pub.sent(".call.start"), "a conexão suporta uma chamada por vez")
}

// 7.3: histórico restrito à empresa e, sem alcance de empresa, aos tickets visíveis.
func TestCallHistory_TenantAndOperatorIsolation(t *testing.T) {
	w := newCallWorld(t)
	mkLog := func(tenant uuid.UUID, id string, ticketID *int, wa int) {
		require.NoError(t, w.db.Create(&models.CallLog{TenantID: tenant, CallID: id, WhatsappID: wa, TicketID: ticketID,
			Direction: "incoming", Status: "ended"}).Error)
	}
	tk2 := models.Ticket{Status: "open", ContactID: w.ticket.ContactID, WhatsappID: w.wa.ID, TenantID: w.tenant}
	require.NoError(t, w.db.Create(&tk2).Error)
	mkLog(w.tenant, "L-MINE", &w.ticket.ID, w.wa.ID)
	mkLog(w.tenant, "L-OTHERS", &tk2.ID, w.wa.ID)
	foreignWA := models.Whatsapp{Name: "F" + w.other.String(), TenantID: w.other, Status: "CONNECTED"}
	require.NoError(t, w.db.Create(&foreignWA).Error)
	mkLog(w.other, "L-FOREIGN", nil, foreignWA.ID)

	agent := w.user(t, "agente", "proprio", w.tenant, "read")
	require.NoError(t, w.db.Model(&models.Ticket{}).Where("id = ?", w.ticket.ID).Update("userId", agent.ID).Error)
	admin := w.user(t, "admin", "tenant", w.tenant, "read")

	ids := func(r *gin.Engine) []string {
		res := do(r, "GET", "/calls", nil)
		require.Equal(t, http.StatusOK, res.Code)
		var out struct {
			Calls []models.CallLog `json:"calls"`
			Total int64            `json:"total"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &out))
		var s []string
		for _, c := range out.Calls {
			s = append(s, c.CallID)
		}
		assert.EqualValues(t, len(s), out.Total)
		return s
	}
	assert.ElementsMatch(t, []string{"L-MINE", "L-OTHERS"}, ids(w.router(admin)), "alcance de empresa vê toda a empresa, nunca outra")
	assert.Equal(t, []string{"L-MINE"}, ids(w.router(agent)), "operador vê só chamadas de tickets que enxerga")

	assert.Equal(t, http.StatusOK, do(w.router(agent), "GET", "/calls/L-MINE", nil).Code)
	assert.Equal(t, http.StatusNotFound, do(w.router(agent), "GET", "/calls/L-OTHERS", nil).Code, "fora do alcance: 404, sem revelar que existe")
	assert.Equal(t, http.StatusNotFound, do(w.router(admin), "GET", "/calls/L-FOREIGN", nil).Code, "outra empresa: 404")
}

func TestCallHistory_FiltersAndPaging(t *testing.T) {
	w := newCallWorld(t)
	for i, st := range []string{"ended", "missed", "ended", "rejected"} {
		require.NoError(t, w.db.Create(&models.CallLog{TenantID: w.tenant, CallID: "H-" + strconv.Itoa(i), WhatsappID: w.wa.ID,
			TicketID: &w.ticket.ID, Direction: "incoming", Status: st}).Error)
	}
	r := w.router(w.user(t, "admin", "tenant", w.tenant, "read"))
	var out struct {
		Calls []models.CallLog `json:"calls"`
		Total int64            `json:"total"`
	}
	res := do(r, "GET", "/calls?status=ended", nil)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &out))
	assert.EqualValues(t, 2, out.Total)

	res = do(r, "GET", "/calls?pageSize=2&page=2", nil)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &out))
	assert.EqualValues(t, 4, out.Total, "o total ignora a página")
	assert.Len(t, out.Calls, 2)
}

func TestCallPause_AcceptsState(t *testing.T) {
	w := newCallWorld(t)
	u := w.user(t, "p", "proprio", w.tenant, "receive")
	assert.Equal(t, http.StatusNoContent, do(w.router(u), "PUT", "/calls/pause", map[string]interface{}{"paused": true}).Code)
}

func TestCallAccept_UnknownIs404AndConflictCodes(t *testing.T) {
	w := newCallWorld(t)
	u := w.user(t, "atende", "proprio", w.tenant, "receive")
	assert.Equal(t, http.StatusNotFound, do(w.router(u), "POST", "/calls/NAO-EXISTE/accept", nil).Code)
	require.NoError(t, w.db.Create(&models.CallLog{TenantID: w.tenant, CallID: "ACC", WhatsappID: w.wa.ID, Direction: "incoming", Status: "active"}).Error)
	res := do(w.router(u), "POST", "/calls/ACC/accept", nil)
	assert.Equal(t, http.StatusConflict, res.Code)
	assert.Equal(t, "ALREADY_ANSWERED", code(res))
}

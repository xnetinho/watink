package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/calls"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	deleted []string
}

func (s *recStore) Upload(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	b, _ := io.ReadAll(r)
	s.mu.Lock()
	s.objects[key] = b
	s.mu.Unlock()
	return nil
}
func (s *recStore) Download(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (s *recStore) PresignedGetURL(_ context.Context, key string, ttl time.Duration) (string, error) {
	return "https://s3.local/" + key + "?ttl=" + ttl.String(), nil
}
func (s *recStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, key)
	delete(s.objects, key)
	return nil
}
func (s *recStore) Describe() map[string]any { return nil }

type recWorld struct {
	*callWorld
	store *recStore
}

func newRecWorld(t *testing.T, withStore bool) *recWorld {
	t.Helper()
	w := newCallWorld(t)
	rw := &recWorld{callWorld: w, store: &recStore{objects: map[string][]byte{}}}
	w.svc = calls.NewService(w.db, repository.NewGORMContactRepo(w.db), repository.NewGORMTicketRepo(w.db),
		repository.NewGORMQueueRepo(w.db), w.pub, nil, nil)
	if withStore {
		w.svc.WithRecording(calls.NewRecording(rw.store, t.TempDir()))
	} else {
		w.svc.WithRecording(calls.NewRecording(nil, t.TempDir()))
	}
	return rw
}

func (w *recWorld) recRouter(u models.User) *gin.Engine {
	r := w.router(u)
	cc := NewCallController(w.svc)
	any2 := auth.RequireAnyPermission([2]string{"calls", "receive"}, [2]string{"calls", "place"})
	r.GET("/calls/recording-config", auth.RequirePermission("calls", "manage"), cc.GetRecordingConfig)
	r.PUT("/calls/recording-config", auth.RequirePermission("calls", "manage"), cc.PutRecordingConfig)
	r.POST("/calls/:id/recording/start", any2, cc.StartRecording)
	r.POST("/calls/:id/recording/stop", any2, cc.StopRecording)
	r.GET("/calls/:id/recording", auth.RequirePermission("calls", "read"), cc.ListenRecording)
	r.DELETE("/calls/:id/recording", auth.RequirePermission("calls", "delete"), cc.DeleteRecording)
	return r
}

func (w *recWorld) recorded(t *testing.T, callID string, ticketID *int) {
	t.Helper()
	key := callID + "/key.mp3"
	w.store.objects[key] = []byte("mp3")
	require.NoError(t, w.db.Create(&models.CallLog{
		TenantID: w.tenant, CallID: callID, WhatsappID: w.wa.ID, TicketID: ticketID, Direction: "incoming", Status: "ended",
		RecordingKey: key, RecordingStatus: "ready",
	}).Error)
}

func accessCount(t *testing.T, w *recWorld, callID, action string) int64 {
	var n int64
	require.NoError(t, w.db.Model(&models.CallRecordingAccess{}).
		Where(`"tenantId" = ? AND "callId" = ? AND action = ?`, w.tenant, callID, action).Count(&n).Error)
	return n
}

// 7.10: permissão, escopo, auditoria e 404 sem revelar existência.
func TestRecordingListen_PermissionScopeAndAudit(t *testing.T) {
	w := newRecWorld(t, true)
	w.recorded(t, "R-MINE", &w.ticket.ID)
	other := models.Ticket{Status: "open", ContactID: w.ticket.ContactID, WhatsappID: w.wa.ID, TenantID: w.tenant}
	require.NoError(t, w.db.Create(&other).Error)
	w.recorded(t, "R-OTHERS", &other.ID)

	agent := w.user(t, "agente", "proprio", w.tenant, "read")
	require.NoError(t, w.db.Model(&models.Ticket{}).Where("id = ?", w.ticket.ID).Update("userId", agent.ID).Error)

	res := do(w.recRouter(agent), "GET", "/calls/R-MINE/recording", nil)
	require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	assert.Contains(t, body["url"], "https://s3.local/R-MINE/key.mp3", "a URL assinada é gerada na leitura")
	assert.EqualValues(t, 300, body["expiresInSeconds"])
	assert.EqualValues(t, 1, accessCount(t, w, "R-MINE", "listen"), "a escuta é registrada")
	var rec models.CallRecordingAccess
	require.NoError(t, w.db.Where(`"callId" = ?`, "R-MINE").First(&rec).Error)
	assert.Equal(t, agent.ID, rec.UserID, "registra QUEM ouviu")
	assert.False(t, rec.At.IsZero(), "e quando")

	res = do(w.recRouter(agent), "GET", "/calls/R-OTHERS/recording", nil)
	assert.Equal(t, http.StatusNotFound, res.Code, "fora do alcance: 404, sem revelar que existe")
	assert.EqualValues(t, 0, accessCount(t, w, "R-OTHERS", "listen"), "tentativa negada não vira escuta")
	assert.Equal(t, http.StatusNotFound, do(w.recRouter(agent), "GET", "/calls/NAO-EXISTE/recording", nil).Code, "mesma resposta de quem não existe")

	noRead := w.user(t, "semleitura", "proprio", w.tenant, "receive")
	assert.Equal(t, http.StatusForbidden, do(w.recRouter(noRead), "GET", "/calls/R-MINE/recording", nil).Code, "sem calls:read")
}

func TestRecordingListen_OtherTenantAndNoRecording(t *testing.T) {
	w := newRecWorld(t, true)
	w.recorded(t, "R-1", &w.ticket.ID)
	intruder := w.user(t, "intruso", "tenant", w.other, "read")
	assert.Equal(t, http.StatusNotFound, do(w.recRouter(intruder), "GET", "/calls/R-1/recording", nil).Code, "outra empresa")

	require.NoError(t, w.db.Create(&models.CallLog{TenantID: w.tenant, CallID: "R-NONE", WhatsappID: w.wa.ID, TicketID: &w.ticket.ID,
		Direction: "incoming", Status: "ended"}).Error)
	admin := w.user(t, "admin", "tenant", w.tenant, "read")
	assert.Equal(t, http.StatusNotFound, do(w.recRouter(admin), "GET", "/calls/R-NONE/recording", nil).Code, "chamada sem gravação")
}

func TestRecordingDelete_RemovesFromStoreAuditsAndMarksHistory(t *testing.T) {
	w := newRecWorld(t, true)
	w.recorded(t, "R-DEL", &w.ticket.ID)
	admin := w.user(t, "admin", "tenant", w.tenant, "delete")

	assert.Equal(t, http.StatusForbidden, do(w.recRouter(w.user(t, "leitor", "proprio", w.tenant, "read")), "DELETE", "/calls/R-DEL/recording", nil).Code, "excluir exige calls:delete")

	res := do(w.recRouter(admin), "DELETE", "/calls/R-DEL/recording", nil)
	require.Equal(t, http.StatusNoContent, res.Code)
	assert.Equal(t, []string{"R-DEL/key.mp3"}, w.store.deleted, "o arquivo sai do armazenamento")
	assert.Empty(t, w.store.objects)
	assert.EqualValues(t, 1, accessCount(t, w, "R-DEL", "delete"), "a exclusão é registrada")

	var l models.CallLog
	require.NoError(t, w.db.Where(`"callId" = ?`, "R-DEL").First(&l).Error)
	assert.Equal(t, "deleted", l.RecordingStatus, "o histórico indica que foi excluída")
	assert.Empty(t, l.RecordingKey)

	assert.Equal(t, http.StatusNotFound, do(w.recRouter(admin), "DELETE", "/calls/R-DEL/recording", nil).Code, "excluir de novo: não há mais")
	assert.Equal(t, http.StatusNotFound, do(w.recRouter(admin), "GET", "/calls/R-DEL/recording", nil).Code, "e não dá mais para ouvir")
}

func TestRecordingDelete_OutOfScopeIs404AndKeepsTheFile(t *testing.T) {
	w := newRecWorld(t, true)
	other := models.Ticket{Status: "open", ContactID: w.ticket.ContactID, WhatsappID: w.wa.ID, TenantID: w.tenant}
	require.NoError(t, w.db.Create(&other).Error)
	w.recorded(t, "R-X", &other.ID)
	agent := w.user(t, "agente", "proprio", w.tenant, "delete")
	assert.Equal(t, http.StatusNotFound, do(w.recRouter(agent), "DELETE", "/calls/R-X/recording", nil).Code)
	assert.Empty(t, w.store.deleted, "nada foi apagado")
	assert.EqualValues(t, 0, accessCount(t, w, "R-X", "delete"))
}

func TestRecordingRoutes_WithoutS3(t *testing.T) {
	w := newRecWorld(t, false)
	w.recorded(t, "R-S3", &w.ticket.ID)
	admin := w.user(t, "admin", "tenant", w.tenant, "read", "delete", "manage")
	res := do(w.recRouter(admin), "GET", "/calls/R-S3/recording", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
	assert.Equal(t, "NO_STORAGE", code(res))
	assert.Equal(t, http.StatusUnprocessableEntity, do(w.recRouter(admin), "DELETE", "/calls/R-S3/recording", nil).Code)
	assert.EqualValues(t, 0, accessCount(t, w, "R-S3", "listen"), "sem armazenamento não há escuta registrada")

	res = do(w.recRouter(admin), "GET", "/calls/recording-config", nil)
	require.Equal(t, http.StatusOK, res.Code)
	var cfg map[string]interface{}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &cfg))
	assert.Equal(t, false, cfg["available"], "a UI recebe que as opções de gravação não são oferecidas")
}

// 7.12: configuração com aceite.
func TestRecordingConfig_RequiresAckToLeaveOff(t *testing.T) {
	w := newRecWorld(t, true)
	mgr := w.user(t, "gerente", "proprio", w.tenant, "manage")
	r := w.recRouter(mgr)

	res := do(r, "GET", "/calls/recording-config", nil)
	require.Equal(t, http.StatusOK, res.Code)
	var cfg calls.RecordingConfig
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &cfg))
	assert.Equal(t, "off", cfg.Mode, "padrão: não grava")
	assert.True(t, cfg.Available)
	assert.Nil(t, cfg.AckBy)

	res = do(r, "PUT", "/calls/recording-config", map[string]interface{}{"mode": "auto"})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code, "sem aceite não muda")
	assert.Equal(t, "ACK_REQUIRED", code(res))
	assert.Equal(t, "off", callsMode(t, w), "e nada foi gravado")

	res = do(r, "PUT", "/calls/recording-config", map[string]interface{}{"mode": "auto", "ack": true})
	require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &cfg))
	assert.Equal(t, "auto", cfg.Mode)
	require.NotNil(t, cfg.AckBy)
	assert.Equal(t, mgr.ID, *cfg.AckBy, "o aceite grava quem aceitou")
	require.NotNil(t, cfg.AckAt)
	assert.WithinDuration(t, time.Now(), *cfg.AckAt, time.Minute, "e quando")
}

func callsMode(t *testing.T, w *recWorld) string {
	var v string
	w.db.Raw(`SELECT value FROM "Settings" WHERE key = 'callRecordingMode' AND "tenantId" = ?`, w.tenant).Scan(&v)
	if v == "" {
		return "off"
	}
	return v
}

func TestRecordingConfig_ChangeBetweenOnModesAndBackToOffNeedNoNewAck(t *testing.T) {
	w := newRecWorld(t, true)
	r := w.recRouter(w.user(t, "gerente", "proprio", w.tenant, "manage"))
	require.Equal(t, http.StatusOK, do(r, "PUT", "/calls/recording-config", map[string]interface{}{"mode": "optional", "ack": true}).Code)
	assert.Equal(t, http.StatusOK, do(r, "PUT", "/calls/recording-config", map[string]interface{}{"mode": "auto"}).Code, "de um modo ligado para outro não pede aceite de novo")
	assert.Equal(t, http.StatusOK, do(r, "PUT", "/calls/recording-config", map[string]interface{}{"mode": "off"}).Code, "desligar nunca exige aceite")
	assert.Equal(t, "off", callsMode(t, w))
	assert.Equal(t, http.StatusUnprocessableEntity, do(r, "PUT", "/calls/recording-config", map[string]interface{}{"mode": "auto"}).Code,
		"voltando de off, o aceite é exigido outra vez")
}

func TestRecordingConfig_BadModeWithoutS3AndPermission(t *testing.T) {
	w := newRecWorld(t, true)
	mgr := w.user(t, "gerente", "proprio", w.tenant, "manage")
	assert.Equal(t, http.StatusBadRequest, do(w.recRouter(mgr), "PUT", "/calls/recording-config", map[string]interface{}{"mode": "sempre", "ack": true}).Code)
	assert.Equal(t, http.StatusBadRequest, do(w.recRouter(mgr), "PUT", "/calls/recording-config", map[string]interface{}{}).Code)

	plain := w.user(t, "comum", "proprio", w.tenant, "read")
	assert.Equal(t, http.StatusForbidden, do(w.recRouter(plain), "PUT", "/calls/recording-config", map[string]interface{}{"mode": "auto", "ack": true}).Code, "sem calls:manage")
	assert.Equal(t, http.StatusForbidden, do(w.recRouter(plain), "GET", "/calls/recording-config", nil).Code)

	noS3 := newRecWorld(t, false)
	m2 := noS3.user(t, "gerente", "proprio", noS3.tenant, "manage")
	res := do(noS3.recRouter(m2), "PUT", "/calls/recording-config", map[string]interface{}{"mode": "auto", "ack": true})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code, "sem armazenamento só 'off' é aceito")
	assert.Equal(t, "NO_STORAGE", code(res))
	assert.Equal(t, http.StatusOK, do(noS3.recRouter(m2), "PUT", "/calls/recording-config", map[string]interface{}{"mode": "off"}).Code)
}

func TestRecordingConfig_IsolatedPerTenant(t *testing.T) {
	w := newRecWorld(t, true)
	r := w.recRouter(w.user(t, "gerente", "proprio", w.tenant, "manage"))
	require.Equal(t, http.StatusOK, do(r, "PUT", "/calls/recording-config", map[string]interface{}{"mode": "auto", "ack": true}).Code)

	var v string
	w.db.Raw(`SELECT value FROM "Settings" WHERE key = 'callRecordingMode' AND "tenantId" = ?`, w.other).Scan(&v)
	assert.Empty(t, v, "o modo de uma empresa não vaza para outra")
	_ = uuid.Nil
}

// Start/stop pela rota: modo off rejeita (422), modo optional aceita.
func TestRecordingStartStop_Routes(t *testing.T) {
	w := newRecWorld(t, true)
	op := w.user(t, "operador", "proprio", w.tenant, "receive")
	r := w.recRouter(op)
	require.NoError(t, w.db.Create(&models.CallLog{TenantID: w.tenant, CallID: "ST-1", WhatsappID: w.wa.ID, TicketID: &w.ticket.ID,
		Direction: "incoming", Status: "active", HandledByUserID: &op.ID}).Error)

	res := do(r, "POST", "/calls/ST-1/recording/start", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code, "modo off (padrão) rejeita")
	assert.Equal(t, "RECORDING_OFF", code(res))

	require.NoError(t, w.db.Create(&models.Setting{Key: "callRecordingMode", TenantID: w.tenant, Value: "optional"}).Error)
	assert.Equal(t, http.StatusNoContent, do(r, "POST", "/calls/ST-1/recording/start", nil).Code)
	assert.Equal(t, http.StatusConflict, do(r, "POST", "/calls/ST-1/recording/start", nil).Code, "já está gravando")
	assert.Equal(t, http.StatusNoContent, do(r, "POST", "/calls/ST-1/recording/stop", nil).Code)
	assert.Equal(t, http.StatusConflict, do(r, "POST", "/calls/ST-1/recording/stop", nil).Code, "já parou")
}

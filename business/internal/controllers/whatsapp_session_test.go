package controllers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// mockChannelSessionRepo implements domain.ChannelSessionRepository for testing.
type mockChannelSessionRepo struct {
	session   *domain.ChannelSession
	sessions  []domain.ChannelSession
	findErr   error
	updateErr error
}

func (m *mockChannelSessionRepo) FindByID(_ context.Context, _ int, _ uuid.UUID) (*domain.ChannelSession, error) {
	return m.session, m.findErr
}
func (m *mockChannelSessionRepo) FindByIDDetail(_ context.Context, _ int, _ uuid.UUID) (*models.Whatsapp, error) {
	panic("not used")
}
func (m *mockChannelSessionRepo) FindAll(_ context.Context, _ uuid.UUID) ([]domain.ChannelSession, error) {
	return m.sessions, m.findErr
}
func (m *mockChannelSessionRepo) Create(_ context.Context, _ *domain.ChannelSession) error {
	return nil
}
func (m *mockChannelSessionRepo) Update(_ context.Context, _ *domain.ChannelSession, _ map[string]interface{}) error {
	return m.updateErr
}
func (m *mockChannelSessionRepo) Delete(_ context.Context, _ int, _ uuid.UUID) error    { return nil }
func (m *mockChannelSessionRepo) ResetDefaultFlag(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockChannelSessionRepo) DeleteWithRelations(_ context.Context, _ int, _ uuid.UUID) error {
	return nil
}

func newSessionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewTestDB(t)
}

func setupSessionGinContext(tenantID uuid.UUID, method, path string, db *gorm.DB) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(method, path, nil)
	c.Request = req
	c.Set("tenantId", tenantID)
	c.Set("alcance", "tenant")
	c.Set("userId", float64(1))
	c.Set("db", db)
	return c, w
}

func TestSessionController_StartSession_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	db := newSessionTestDB(t)

	repo := &mockChannelSessionRepo{session: nil, findErr: errors.New("not found")}
	ctrl := NewSessionController(repo, nil, nil)

	c, w := setupSessionGinContext(tenantID, "POST", "/whatsappsession/99", db)
	c.Params = gin.Params{{Key: "whatsappId", Value: "99"}}

	ctrl.StartSession(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSessionController_StartSession_NoTenantID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockChannelSessionRepo{}
	ctrl := NewSessionController(repo, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("POST", "/whatsappsession/1", nil)
	c.Request = req
	// no tenantId set — GetScoped returns 400 bad request

	ctrl.StartSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSessionController_StopSession_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	db := newSessionTestDB(t)

	repo := &mockChannelSessionRepo{session: nil, findErr: nil}
	ctrl := NewSessionController(repo, nil, nil)

	c, w := setupSessionGinContext(tenantID, "DELETE", "/whatsappsession/99", db)
	c.Params = gin.Params{{Key: "whatsappId", Value: "99"}}

	ctrl.StopSession(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSessionController_RestartAllSessions_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	db := newSessionTestDB(t)

	repo := &mockChannelSessionRepo{sessions: []domain.ChannelSession{}}
	ctrl := NewSessionController(repo, nil, nil)

	c, w := setupSessionGinContext(tenantID, "POST", "/whatsappsession/all", db)

	ctrl.RestartAllSessions(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Restarting")
}

func TestSessionController_RestartAllSessions_FetchError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	db := newSessionTestDB(t)

	repo := &mockChannelSessionRepo{findErr: errors.New("db error")}
	ctrl := NewSessionController(repo, nil, nil)

	c, w := setupSessionGinContext(tenantID, "POST", "/whatsappsession/all", db)

	ctrl.RestartAllSessions(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// noopCommandPublisher/busyLockRedis são dublês mínimos só para exercitar o controller com um
// WhatsAppSessionService REAL (não mockado), provando o status HTTP de ponta a ponta — um mock do
// service esconderia justamente o bug deste caso (500 em vez de 409).
type noopCommandPublisher struct{}

func (noopCommandPublisher) PublishCommand(string, interface{}) error { return nil }

type busyLockRedis struct{}

func (busyLockRedis) SetLock(string, string, time.Duration) (bool, error) { return false, nil }
func (busyLockRedis) DelLock(string) error                                { return nil }
func (busyLockRedis) Subscribe(context.Context, string) *redis.PubSub     { return nil }
func (busyLockRedis) Publish(context.Context, string, interface{}) error  { return nil }
func (busyLockRedis) Ping(context.Context) error                          { return nil }
func (busyLockRedis) Get(context.Context, string) (string, error)         { return "", nil }

// TestSessionController_StartSession_LockBusy_Returns409NotInternalError é o teste de regressão do
// bug real: clique duplo (ou outra aba) em "Iniciar sessão" batia no lock Redis ocupado e o
// controller devolvia 500 "Internal server error" — porque o erro de lock não era um
// *utils.FriendlyError, e RespondWithFriendlyOrInternalError caía no genérico. O caso é um
// conflito esperado, não uma falha do servidor: a resposta certa é 409.
func TestSessionController_StartSession_LockBusy_Returns409NotInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	db := newSessionTestDB(t)

	sess := &domain.ChannelSession{ID: 1, Name: "zap-teste", TenantID: tenantID, EngineType: "whatsmeow"}
	repo := &mockChannelSessionRepo{session: sess}
	broadcast := services.NewRedisBroadcast(busyLockRedis{}, nil)
	wss := services.NewWhatsAppSessionService(db, noopCommandPublisher{}, busyLockRedis{}, broadcast)
	ctrl := NewSessionController(repo, broadcast, wss)

	c, w := setupSessionGinContext(tenantID, "POST", "/whatsappsession/1", db)
	c.Params = gin.Params{{Key: "whatsappId", Value: "1"}}

	ctrl.StartSession(c)

	assert.Equal(t, http.StatusConflict, w.Code, "lock ocupado é conflito (409), não erro de servidor (500): body=%s", w.Body.String())
	assert.NotContains(t, w.Body.String(), "Internal server error")
}

// TestChannelSessionToModel_PreservesWid prova o invariante: channelSessionToModel
// (usado por Start/Stop/RestartAllSessions antes de chamar o sessionService) NÃO
// pode perder o Wid. Sem ele, resolveDeviceStore no engine não acha o device
// existente, cria um device NOVO e desloga a conta WhatsApp real no próximo
// Stop+Start — reproduzido em runtime ao testar o fix B1 (SetProxyAddress).
func TestChannelSessionToModel_PreservesWid(t *testing.T) {
	const wid = "558597964683:2@s.whatsapp.net"
	s := &domain.ChannelSession{ID: 3, Name: "zap-4683", Wid: wid}
	m := channelSessionToModel(s)
	assert.Equal(t, wid, m.Wid, "channelSessionToModel deve preservar o Wid")
}

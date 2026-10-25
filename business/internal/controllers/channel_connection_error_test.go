package controllers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/application"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/routes"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type revocableChannel struct {
	tokenRevoked bool
}

func (m *revocableChannel) Kind() domain.ChannelKind { return "telegram" }
func (m *revocableChannel) Capabilities() domain.Capabilities {
	return domain.Capabilities{InitiateConversation: false, MaxTextBytes: 4096}
}
func (m *revocableChannel) Connect(ctx context.Context, conn *models.ChannelConnection, input domain.ConnectInput) (domain.ConnectResult, error) {
	if m.tokenRevoked || input.Token == "revoked-token" {
		return domain.ConnectResult{Status: "DISCONNECTED", Connected: false}, errors.New("Unauthorized: token was revoked")
	}
	return domain.ConnectResult{Status: "CONNECTED", Connected: true}, nil
}
func (m *revocableChannel) Disconnect(ctx context.Context, conn *models.ChannelConnection) error {
	return nil
}
func (m *revocableChannel) Send(ctx context.Context, conn *models.ChannelConnection, msg domain.OutboundMessage) (domain.SendResult, error) {
	if m.tokenRevoked {
		return domain.SendResult{}, errors.New("Forbidden: bot was blocked or token revoked")
	}
	return domain.SendResult{ExternalMessageID: "msg-1", Status: "sent", Timestamp: time.Now()}, nil
}
func (m *revocableChannel) ResolveAddress(ctx context.Context, conn *models.ChannelConnection, contact *models.Contact) (string, error) {
	return contact.Number, nil
}
func (m *revocableChannel) ParseInbound(ctx context.Context, conn *models.ChannelConnection, raw []byte, headers map[string]string) ([]domain.InboundMessage, error) {
	return nil, nil
}

func TestChannelConnection_LastErrorVisibleOnTokenRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("PROXY_ENC_KEY", "01234567890123456789012345678901")

	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "Empresa Error", Status: "active"}).Error)

	cargo := models.Cargo{Name: "Admin", TenantID: tenantID}
	require.NoError(t, db.Create(&cargo).Error)
	user := models.User{Name: "Admin", Email: "admin@err.io", TenantID: tenantID, CargoID: &cargo.ID, Alcance: "tenant"}
	require.NoError(t, db.Create(&user).Error)

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": user.ID, "tenantId": tenantID.String(), "alcance": "tenant", "username": user.Name, "exp": 4102444800,
	}).SignedString([]byte("test-secret"))
	require.NoError(t, err)

	r := gin.New()
	routes.SetupRoutes(r.Group("/api/v1"), nil, application.NewContainer(db, nil, nil, nil), nil, nil, nil, nil)

	ch := &revocableChannel{}
	domain.DefaultChannelRegistry().Register(ch)

	// 1. Criar conexão com token revogado -> deve persistir LastError e responder com erro
	createBody := `{"name":"Bot Revogado","kind":"telegram","token":"revoked-token","mode":"polling"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/connections", strings.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var created models.ChannelConnection
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "DISCONNECTED", created.Status)
	assert.Contains(t, created.LastError, "Unauthorized: token was revoked")

	// 2. GET /channels/connections deve expor LastError para que o frontend exiba no card
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/channels/connections", nil)
	reqGet.Header.Set("Authorization", "Bearer "+tok)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	require.Equal(t, http.StatusOK, wGet.Code)
	var list []models.ChannelConnection
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &list))
	require.Len(t, list, 1)
	assert.Equal(t, "DISCONNECTED", list[0].Status)
	assert.Contains(t, list[0].LastError, "Unauthorized: token was revoked")

	// 3. Atualizar com token válido -> reconecta e limpa LastError
	ch.tokenRevoked = false
	updateBody := `{"token":"valid-token"}`
	reqPut := httptest.NewRequest(http.MethodPut, "/api/v1/channels/connections/"+itoa(created.ID), strings.NewReader(updateBody))
	reqPut.Header.Set("Authorization", "Bearer "+tok)
	reqPut.Header.Set("Content-Type", "application/json")
	wPut := httptest.NewRecorder()
	r.ServeHTTP(wPut, reqPut)

	require.Equal(t, http.StatusOK, wPut.Code)
	var updated models.ChannelConnection
	require.NoError(t, json.Unmarshal(wPut.Body.Bytes(), &updated))
	assert.Equal(t, "CONNECTED", updated.Status)
	assert.Empty(t, updated.LastError)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

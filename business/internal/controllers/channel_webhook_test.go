package controllers_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/controllers"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type webhookTestChannel struct {
	kind domain.ChannelKind
}

func (w *webhookTestChannel) Kind() domain.ChannelKind         { return w.kind }
func (w *webhookTestChannel) Capabilities() domain.Capabilities { return domain.Capabilities{} }
func (w *webhookTestChannel) Connect(_ context.Context, _ *models.ChannelConnection, _ domain.ConnectInput) (domain.ConnectResult, error) {
	return domain.ConnectResult{}, nil
}
func (w *webhookTestChannel) Disconnect(_ context.Context, _ *models.ChannelConnection) error {
	return nil
}
func (w *webhookTestChannel) Send(_ context.Context, _ *models.ChannelConnection, _ domain.OutboundMessage) (domain.SendResult, error) {
	return domain.SendResult{}, nil
}
func (w *webhookTestChannel) ParseInbound(_ context.Context, _ *models.ChannelConnection, raw []byte, _ map[string]string) ([]domain.InboundMessage, error) {
	return []domain.InboundMessage{
		{
			ExternalMessageID: "wh-msg-1",
			ConversationKey:   "user-1",
			Sender: domain.InboundSender{
				ExternalID: "user-1",
				Name:       "Webhook User",
			},
			Body:      string(raw),
			Timestamp: time.Now(),
		},
	}, nil
}

func TestChannelWebhookController_SecretAndAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)

	tenantID := uuid.New()
	tenant := models.Tenant{ID: tenantID, Name: "Tenant Webhook"}
	require.NoError(t, db.Create(&tenant).Error)

	secretKey := "my-super-secret-key-123"
	conn := models.ChannelConnection{
		TenantID:      tenantID,
		Kind:          string(domain.ChannelKindTelegram),
		Name:          "Bot Telegram Seguro",
		Status:        "CONNECTED",
		WebhookSecret: secretKey,
	}
	require.NoError(t, db.Create(&conn).Error)

	// Registra o adaptador de canal para o teste
	domain.DefaultChannelRegistry().Register(&webhookTestChannel{kind: domain.ChannelKindTelegram})

	wc := controllers.NewChannelWebhookController(db, nil)
	r := gin.New()
	r.POST("/webhooks/channels/:kind/:connectionId", wc.HandleWebhook)

	rawBody := []byte(`{"text":"teste webhook"}`)

	// 1. Conexão inexistente -> 404
	w404 := httptest.NewRecorder()
	req404, _ := http.NewRequest("POST", "/webhooks/channels/telegram/99999", bytes.NewBuffer(rawBody))
	r.ServeHTTP(w404, req404)
	assert.Equal(t, http.StatusNotFound, w404.Code)

	// 2. Conexão existente, mas sem assinatura ou assinatura errada -> 401
	w401 := httptest.NewRecorder()
	req401, _ := http.NewRequest("POST", "/webhooks/channels/telegram/"+strconv.Itoa(conn.ID), bytes.NewBuffer(rawBody))
	req401.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-token")
	r.ServeHTTP(w401, req401)
	assert.Equal(t, http.StatusUnauthorized, w401.Code)

	// 3. Conexão existente com token de header correto do Telegram -> 200 OK
	w200Tg := httptest.NewRecorder()
	req200Tg, _ := http.NewRequest("POST", "/webhooks/channels/telegram/"+strconv.Itoa(conn.ID), bytes.NewBuffer(rawBody))
	req200Tg.Header.Set("X-Telegram-Bot-Api-Secret-Token", secretKey)
	r.ServeHTTP(w200Tg, req200Tg)
	assert.Equal(t, http.StatusOK, w200Tg.Code)

	// 4. Conexão existente com HMAC-SHA256 correto (padrão Meta/Instagram) -> 200 OK
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(rawBody)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	w200Meta := httptest.NewRecorder()
	req200Meta, _ := http.NewRequest("POST", "/webhooks/channels/telegram/"+strconv.Itoa(conn.ID), bytes.NewBuffer(rawBody))
	req200Meta.Header.Set("X-Hub-Signature-256", "sha256="+expectedSig)
	r.ServeHTTP(w200Meta, req200Meta)
	assert.Equal(t, http.StatusOK, w200Meta.Code)
}

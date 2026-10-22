package controllers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ChannelWebhookController gerencia o recebimento de webhooks externos de canais agnósticos (D6 / ADR 0033).
type ChannelWebhookController struct {
	db             *gorm.DB
	inboundService *services.ChannelInboundService
}

func NewChannelWebhookController(db *gorm.DB, inbound *services.ChannelInboundService) *ChannelWebhookController {
	return &ChannelWebhookController{
		db:             db,
		inboundService: inbound,
	}
}

// HandleWebhook processa o webhook recebido de um canal externo.
//
// @Summary      Receber webhook de canal
// @Tags         channels
// @Accept       json
// @Produce      json
// @Param        kind          path  string  true  "Tipo do canal (telegram, mercadolivre, instagram, etc.)"
// @Param        connectionId  path  int     true  "ID da ChannelConnection"
// @Success      200           {object}  map[string]interface{}
// @Failure      401           {object}  map[string]string
// @Failure      404           {object}  map[string]string
// @Router       /webhooks/channels/{kind}/{connectionId} [post]
func (wc *ChannelWebhookController) HandleWebhook(c *gin.Context) {
	kind := c.Param("kind")
	connIDStr := c.Param("connectionId")
	connID, err := strconv.Atoi(connIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	// 1. Busca a conexão pelo ID e kind
	var conn models.ChannelConnection
	if err := wc.db.Session(&gorm.Session{NewDB: true}).
		Where("id = ? AND kind = ?", connID, kind).
		First(&conn).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	// 2. Lê os bytes do payload
	raw, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
		return
	}

	// 3. Validação de assinatura / segredo por conexão (fail-closed, sem vazar metadados)
	if conn.WebhookSecret != "" {
		if !validateWebhookSecret(c, raw, conn.WebhookSecret) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_signature"})
			return
		}
	}

	// 4. Resolve o adaptador de canal registrado
	channelAdapter, err := domain.DefaultChannelRegistry().Get(domain.ChannelKind(kind))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel_not_supported"})
		return
	}

	// 5. Converte cabeçalhos HTTP
	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// 6. Faz o parse das mensagens do canal
	inboundMsgs, err := channelAdapter.ParseInbound(c.Request.Context(), &conn, raw, headers)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parse_failed", "details": err.Error()})
		return
	}

	// 7. Encaminha as mensagens para o serviço unificado de entrada
	processed := 0
	failed := 0
	if wc.inboundService != nil {
		for _, msg := range inboundMsgs {
			msg.TenantID = conn.TenantID
			msg.ChannelConnectionID = conn.ID
			msg.Channel = domain.ChannelKind(kind)
			if _, err := wc.inboundService.Receive(c.Request.Context(), msg); err != nil {
				log.Printf("[ChannelWebhook] receive error (conn %d, msg %s): %v", conn.ID, msg.ExternalMessageID, err)
				failed++
				continue
			}
			processed++
		}
	}

	// Falha de processamento responde 5xx: o canal reentrega o update (o Telegram
	// só considera entregue em 2xx), em vez de perder a mensagem silenciosamente.
	if failed > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "received": processed, "failed": failed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "received": processed})
}

// validateWebhookSecret verifica assinaturas HMAC-SHA256 ou tokens de header.
func validateWebhookSecret(c *gin.Context, raw []byte, secret string) bool {
	// Telegram: X-Telegram-Bot-Api-Secret-Token
	if tgSecret := c.GetHeader("X-Telegram-Bot-Api-Secret-Token"); tgSecret != "" {
		return hmac.Equal([]byte(tgSecret), []byte(secret))
	}

	// Meta / Instagram / TikTok: X-Hub-Signature-256 (sha256=<hex>)
	if hubSig := c.GetHeader("X-Hub-Signature-256"); hubSig != "" {
		expectedSig := signHMACSHA256(raw, []byte(secret))
		cleanHubSig := strings.TrimPrefix(hubSig, "sha256=")
		return hmac.Equal([]byte(cleanHubSig), []byte(expectedSig))
	}

	// Header genérico: X-Webhook-Secret
	if genericSecret := c.GetHeader("X-Webhook-Secret"); genericSecret != "" {
		return hmac.Equal([]byte(genericSecret), []byte(secret))
	}

	return false
}

func signHMACSHA256(message, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}

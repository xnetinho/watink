package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/cryptobox"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChannelConnectionController struct{}

func NewChannelConnectionController() *ChannelConnectionController {
	return &ChannelConnectionController{}
}

// ListConnections lista todas as conexões de canais do tenant.
func (cc *ChannelConnectionController) ListConnections(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Whatsapps")
	if !ok {
		return
	}

	var conns []models.ChannelConnection
	if err := db.Where("\"tenantId\" = ?", tenantID).Find(&conns).Error; err != nil {
		utils.RespondWithInternalError(c, err, "ListChannelConnections")
		return
	}

	c.JSON(http.StatusOK, conns)
}

// CreateConnection cria e conecta uma nova ChannelConnection (ex.: Telegram token, com webhook/polling).
func (cc *ChannelConnectionController) CreateConnection(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Whatsapps")
	if !ok {
		return
	}

	var input struct {
		Kind  string `json:"kind" binding:"required"`
		Name  string `json:"name" binding:"required"`
		Token string `json:"token" binding:"required"`
		Mode  string `json:"mode"` // "webhook" (padrão) ou "polling"
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}

	if !cryptobox.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": cryptobox.ErrNotConfigured.Error()})
		return
	}

	encToken, err := cryptobox.Encrypt(input.Token)
	if err != nil {
		utils.RespondWithInternalError(c, err, "EncryptChannelCredentials")
		return
	}

	conn := models.ChannelConnection{
		TenantID:       tenantID,
		Kind:           input.Kind,
		Name:           input.Name,
		CredentialsEnc: encToken,
		Status:         "DISCONNECTED",
	}

	// Cria o registro no banco primeiro para obter o Connection ID (fundamental para a URL de webhook do Telegram)
	if err := db.Create(&conn).Error; err != nil {
		utils.RespondWithInternalError(c, err, "CreateChannelConnection")
		return
	}

	channelAdapter, err := domain.DefaultChannelRegistry().Get(domain.ChannelKind(input.Kind))
	if err == nil {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

		connectInput := domain.ConnectInput{
			Token: input.Token,
			Params: map[string]string{
				"mode":    input.Mode,
				"baseUrl": baseURL,
			},
		}

		if _, connErr := channelAdapter.Connect(c.Request.Context(), &conn, connectInput); connErr != nil {
			conn.Status = "DISCONNECTED"
			conn.LastError = connErr.Error()
		} else {
			conn.Status = "CONNECTED"
			conn.LastError = ""
		}
		_ = db.Save(&conn).Error
	}

	c.JSON(http.StatusOK, conn)
}

// DeleteConnection desconecta e remove uma ChannelConnection.
func (cc *ChannelConnectionController) DeleteConnection(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Whatsapps")
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection id"})
		return
	}

	var conn models.ChannelConnection
	if err := db.Where("id = ? AND \"tenantId\" = ?", id, tenantID).First(&conn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
			return
		}
		utils.RespondWithInternalError(c, err, "DeleteChannelConnection")
		return
	}

	if channelAdapter, err := domain.DefaultChannelRegistry().Get(domain.ChannelKind(conn.Kind)); err == nil {
		_ = channelAdapter.Disconnect(c.Request.Context(), &conn)
	}

	if err := db.Delete(&conn).Error; err != nil {
		utils.RespondWithInternalError(c, err, "DeleteChannelConnection")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

package controllers

import (
	"context"
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

// channelBaseURL monta a URL pública de callback a partir da requisição atual,
// para que o webhook funcione em qualquer domínio (multitenant, sem URL fixa).
func channelBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

// connectAndSave registra o resultado do Connect (status, config e erro) na conexão.
func connectAndSave(ctx context.Context, db *gorm.DB, conn *models.ChannelConnection, token, mode, baseURL string) {
	adapter, err := domain.DefaultChannelRegistry().Get(domain.ChannelKind(conn.Kind))
	if err != nil {
		conn.Status = "DISCONNECTED"
		conn.LastError = err.Error()
		_ = db.Save(conn).Error
		return
	}

	_, connErr := adapter.Connect(ctx, conn, domain.ConnectInput{
		Token: token,
		Params: map[string]string{
			"mode":    mode,
			"baseUrl": baseURL,
		},
	})
	if connErr != nil {
		conn.Status = "DISCONNECTED"
		conn.LastError = connErr.Error()
	} else {
		conn.Status = "CONNECTED"
		conn.LastError = ""
	}
	_ = db.Save(conn).Error
}

type ChannelConnectionController struct{}

func NewChannelConnectionController() *ChannelConnectionController {
	return &ChannelConnectionController{}
}

// ListConnections lista todas as conexões de canais do tenant.
func (cc *ChannelConnectionController) ListConnections(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "ChannelConnections")
	if !ok {
		return
	}

	var conns []models.ChannelConnection
	writeDB := db.Session(&gorm.Session{NewDB: true})
	if err := writeDB.Where("\"tenantId\" = ?", tenantID).Find(&conns).Error; err != nil {
		utils.RespondWithInternalError(c, err, "ListChannelConnections")
		return
	}

	c.JSON(http.StatusOK, conns)
}

// CreateConnection cria e conecta uma nova ChannelConnection (ex.: Telegram token, com webhook/polling).
func (cc *ChannelConnectionController) CreateConnection(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "ChannelConnections")
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

	writeDB := db.Session(&gorm.Session{NewDB: true})
	// Cria o registro no banco primeiro para obter o Connection ID (fundamental para a URL de webhook do Telegram)
	if err := writeDB.Create(&conn).Error; err != nil {
		utils.RespondWithInternalError(c, err, "CreateChannelConnection")
		return
	}

	connectAndSave(c.Request.Context(), writeDB, &conn, input.Token, input.Mode, channelBaseURL(c))

	c.JSON(http.StatusOK, conn)
}

// UpdateConnection edita uma conexão existente (nome, token e modo) e a reconecta.
// Token vazio mantém o segredo atual.
func (cc *ChannelConnectionController) UpdateConnection(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "ChannelConnections")
	if !ok {
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection id"})
		return
	}

	var input struct {
		Name  string `json:"name"`
		Token string `json:"token"`
		Mode  string `json:"mode"` // vazio mantém o modo atual
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}

	var conn models.ChannelConnection
	writeDB := db.Session(&gorm.Session{NewDB: true})
	if err := writeDB.Where("id = ? AND \"tenantId\" = ?", id, tenantID).First(&conn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
			return
		}
		utils.RespondWithInternalError(c, err, "UpdateChannelConnection")
		return
	}

	if input.Name != "" {
		conn.Name = input.Name
	}

	if input.Token != "" {
		if !cryptobox.IsConfigured() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": cryptobox.ErrNotConfigured.Error()})
			return
		}
		encToken, err := cryptobox.Encrypt(input.Token)
		if err != nil {
			utils.RespondWithInternalError(c, err, "EncryptChannelCredentials")
			return
		}
		conn.CredentialsEnc = encToken
	}

	// Persiste a edição antes de reconectar: se a validação do token falhar,
	// nome/modo escolhidos não são perdidos.
	if err := writeDB.Save(&conn).Error; err != nil {
		utils.RespondWithInternalError(c, err, "UpdateChannelConnection")
		return
	}

	connectAndSave(c.Request.Context(), writeDB, &conn, input.Token, input.Mode, channelBaseURL(c))

	c.JSON(http.StatusOK, conn)
}

// DeleteConnection desconecta e remove uma ChannelConnection.
func (cc *ChannelConnectionController) DeleteConnection(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "ChannelConnections")
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
	writeDB := db.Session(&gorm.Session{NewDB: true})
	if err := writeDB.Where("id = ? AND \"tenantId\" = ?", id, tenantID).First(&conn).Error; err != nil {
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

	if err := writeDB.Delete(&conn).Error; err != nil {
		utils.RespondWithInternalError(c, err, "DeleteChannelConnection")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

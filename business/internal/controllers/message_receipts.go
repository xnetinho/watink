package controllers

import (
	"net/http"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListMessageReceipts devolve os dados de entrega e leitura de uma mensagem: por participante, quando recebeu, leu e ouviu.
//
// @Summary      Dados da mensagem (recibos de entrega e leitura)
// @Tags         messages
// @Produce      json
// @Param        messageId  path  string  true  "ID da mensagem"
// @Success      200        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]string
// @Security     BearerAuth
// @Router       /message/{messageId}/receipts [get]
func (mc *MessageController) ListMessageReceipts(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Messages")
	if !ok {
		return
	}
	var msg models.Message
	if err := db.Session(&gorm.Session{NewDB: true}).Select("id").
		Where(`id = ? AND "tenantId" = ?`, c.Param("messageId"), tenantID).First(&msg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	rows, err := services.MessageReceipts(c.Request.Context(), db.Session(&gorm.Session{NewDB: true}), tenantID, msg.ID)
	if err != nil {
		utils.RespondWithInternalError(c, err, "ListMessageReceipts")
		return
	}
	c.JSON(http.StatusOK, gin.H{"receipts": rows})
}

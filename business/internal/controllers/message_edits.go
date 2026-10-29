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

// ListMessageEdits devolve todas as versões de uma mensagem editada, da original (versão 1) à mais recente.
//
// @Summary      Histórico de edições da mensagem
// @Tags         messages
// @Produce      json
// @Param        messageId  path  string  true  "ID da mensagem"
// @Success      200        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]string
// @Security     BearerAuth
// @Router       /message/{messageId}/edits [get]
func (mc *MessageController) ListMessageEdits(c *gin.Context) {
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
	edits, err := services.MessageEdits(c.Request.Context(), db.Session(&gorm.Session{NewDB: true}), tenantID, msg.ID)
	if err != nil {
		utils.RespondWithInternalError(c, err, "ListMessageEdits")
		return
	}
	c.JSON(http.StatusOK, gin.H{"edits": edits})
}

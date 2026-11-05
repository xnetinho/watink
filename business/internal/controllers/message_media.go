package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
)

// DownloadMedia triggers an on-demand download of a pending media message. The
// engine stored a serialized media proto on the message's dataJson at receipt
// time; here we hand it back to the engine, which downloads the bytes and emits
// a "message.media" event that updates the message with the stored media URL.
// @Summary      Baixar mídia sob demanda
// @Tags         messages
// @Produce      json
// @Param        messageId  path      string  true  "ID da mensagem"
// @Success      202        {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /media/{messageId}/download [post]
func (mc *MessageController) DownloadMedia(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Messages")
	if !ok {
		return
	}

	messageID := c.Param("messageId")

	// Load the message together with its ticket in a single query. Reusing the
	// same *gorm.DB for two separate First() calls would accumulate the WHERE
	// conditions (id = msgID AND id = ticketID), which never matches.
	var msg models.Message
	if err := db.Preload("Ticket").Where("id = ? AND \"tenantId\" = ?", messageID, tenantID).First(&msg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	if msg.Ticket.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	var data struct {
		MediaProto string `json:"mediaProto"`
	}
	_ = json.Unmarshal([]byte(msg.DataJson), &data)
	if data.MediaProto == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "media is not downloadable"})
		return
	}

	conn, ch, found, cerr := channelOf(db, tenantID, msg.Ticket)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Connection not found"})
		return
	}
	downloader, can := ch.(domain.MediaDownloader)
	if cerr != nil || !can {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "media is not downloadable"})
		return
	}
	if err := downloader.DownloadMedia(c.Request.Context(), &conn, msg.ID, msg.MediaType, data.MediaProto); err != nil {
		utils.RespondWithInternalError(c, err, "DownloadMedia")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "downloading"})
}

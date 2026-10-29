package controllers

import (
	"net/http"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ResendMessage reenvia, no mesmo ticket, uma mensagem que falhou (ack 5), com o mesmo conteúdo: texto, mídia, voz ou
// arquivo de áudio, legenda e citação. A falha antiga continua no histórico (é o registro de que houve falha) e a nova
// aparece logo abaixo. O chat não sabe remover uma mensagem, e marcá-la como apagada mostraria "mensagem apagada".
//
// Só vale para ack 5. Uma mensagem em ack 1 (celular do contato desligado) ou 2+ está bem: reenviá-la duplicaria.
//
// @Summary      Reenviar mensagem que falhou
// @Tags         messages
// @Produce      json
// @Param        messageId  path  string  true  "ID da mensagem"
// @Success      200        {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /message/{messageId}/resend [post]
func (mc *MessageController) ResendMessage(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Messages")
	if !ok {
		return
	}

	var src models.Message
	if err := db.Session(&gorm.Session{NewDB: true}).Preload("Ticket").Preload("Ticket.Contact").
		Where(`id = ? AND "tenantId" = ?`, c.Param("messageId"), tenantID).First(&src).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	if !src.FromMe || src.IsDeleted {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "message_not_resendable"})
		return
	}
	if src.Ack != ackFailedMessage {
		// Em ack 1 o celular do contato pode só estar desligado; em 2+ já chegou. Reenviar duplicaria.
		c.JSON(http.StatusConflict, gin.H{"error": "message_not_failed"})
		return
	}
	if !forwardableMedia[src.MediaType] {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "message_not_resendable"})
		return
	}
	if src.MediaType != "" && src.MediaType != "chat" && src.MediaUrl == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "media_unavailable"})
		return
	}

	ticket := src.Ticket
	if ticket.Contact.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}
	if p := resolveTicketReplyPolicy(db, &ticket); !p.CanReply {
		c.JSON(http.StatusConflict, gin.H{"error": "reply_not_allowed", "code": string(p.Reason), "message": p.Message})
		return
	}

	quotedID := ""
	if src.QuotedMsgID != nil {
		quotedID = *src.QuotedMsgID
	}
	voice := dataFieldBool(src.DataJson, "voice")
	res, viaChannel, fail := mc.deliver(c.Request.Context(), db, tenantID, ticket, outboundInput{
		Body: src.Body, MediaType: src.MediaType, MediaURL: src.MediaUrl,
		MimeType: dataFieldString(src.DataJson, "mimetype"), QuotedID: quotedID,
		Voice:    voice && src.MediaType == "audio",
		FileName: dataFieldString(src.DataJson, "fileName"),
	})
	if fail != nil {
		fail.respond(c)
		return
	}

	if viaChannel {
		c.JSON(http.StatusOK, res)
		return
	}
	mc.announce(db, tenantID, res)
	c.JSON(http.StatusOK, gin.H{"message": "Message sent", "messageId": res.ID})
}

const ackFailedMessage = 5

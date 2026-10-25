package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ReactToMessage envia ou remove uma reação emoji a uma mensagem WhatsApp existente.
//
// @Summary      Reagir a mensagem
// @Description  Envia uma reação com emoji para a mensagem informada. Emoji vazio remove a reação anterior.
// @Tags         messages
// @Produce      json
// @Param        messageId  path      string                  true  "ID da mensagem"
// @Param        body       body      map[string]interface{}  true  "Emoji da reação (vazio remove)"
// @Success      200        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]string
// @Security     BearerAuth
// @Router       /message/{messageId}/react [post]
func (mc *MessageController) ReactToMessage(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Messages")
	if !ok {
		return
	}

	targetMessageID := c.Param("messageId")

	var input struct {
		Reaction string `json:"reaction"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}

	var target models.Message
	if err := db.Preload("Ticket").Preload("Ticket.Contact").
		Where("id = ? AND \"tenantId\" = ?", targetMessageID, tenantID).
		First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	contact := target.Ticket.Contact
	if contact.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	var whatsapp models.Whatsapp
	if err := db.Session(&gorm.Session{NewDB: true}).
		Where("id = ? AND \"tenantId\" = ?", target.Ticket.WhatsappID, tenantID).First(&whatsapp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Connection not found"})
		return
	}

	command := map[string]interface{}{
		"type": "message.react",
		"payload": map[string]interface{}{
			"sessionId":    target.Ticket.WhatsappID,
			"messageId":    newWAMessageID(),
			"to":           contactJID(contact),
			"targetMsgId":  targetMessageID,
			"targetFromMe": target.FromMe,
			"reaction":     input.Reaction,
		},
	}

	routingKey := wbot.Command(tenantID.String(), strconv.Itoa(target.Ticket.WhatsappID), wbot.CommandMessageReact)
	if err := mc.rabbit.PublishCommand(routingKey, command); err != nil {
		utils.RespondWithInternalError(c, err, "ReactToMessage")
		return
	}

	// Optimistic local update: a reaction WE send doesn't reliably echo back
	// through whatsmeow's *events.Message stream the way inbound reactions
	// do, so we merge/broadcast it here immediately.
	writeDB := db.Session(&gorm.Session{NewDB: true})
	upsertOwnReaction(writeDB, &target, "bot", input.Reaction)
	mc.broadcast.EmitToRoom("/", "chat:"+strconv.Itoa(target.TicketID), "appMessage", map[string]interface{}{
		"action":  "update",
		"message": target,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Reaction sent"})
}

// upsertOwnReaction merges the bot's own reaction into target.Reactions
// (removing it when reaction is empty) and persists the update.
func upsertOwnReaction(db *gorm.DB, target *models.Message, sender, reaction string) {
	var reactions []map[string]interface{}
	if target.Reactions != "" && target.Reactions != "[]" {
		_ = json.Unmarshal([]byte(target.Reactions), &reactions)
	}

	updated := false
	for i, r := range reactions {
		if s, ok := r["sender"].(string); ok && s == sender {
			if reaction == "" {
				reactions = append(reactions[:i], reactions[i+1:]...)
			} else {
				reactions[i]["reaction"] = reaction
			}
			updated = true
			break
		}
	}
	if !updated && reaction != "" {
		reactions = append(reactions, map[string]interface{}{
			"sender":   sender,
			"reaction": reaction,
			"fromMe":   true,
		})
	}

	reactionsJSON, err := json.Marshal(reactions)
	if err != nil {
		return
	}
	target.Reactions = string(reactionsJSON)
	db.Model(&models.Message{}).Where("id = ?", target.ID).Update("reactions", target.Reactions)
}

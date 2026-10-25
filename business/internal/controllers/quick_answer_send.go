package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/flow"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Send envia uma resposta rápida em um ticket aberto.
//
// @Summary      Enviar resposta rápida
// @Tags         quick-answers
// @Accept       json
// @Produce      json
// @Param        quickAnswerId  path      int                     true  "ID da resposta"
// @Param        body           body      map[string]interface{}  true  "ticketId e variables opcionais"
// @Success      200            {object}  map[string]string
// @Security     BearerAuth
// @Router       /quickAnswers/{quickAnswerId}/send [post]
func (qac *QuickAnswerController) Send(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "QuickAnswers")
	if !ok {
		return
	}
	qaID := c.Param("quickAnswerId")

	var input struct {
		TicketID  int               `json:"ticketId"`
		Variables map[string]string `json:"variables"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}
	if input.TicketID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticketId is required"})
		return
	}

	var qa models.QuickAnswer
	if err := db.Where("id = ? AND \"tenantId\" = ?", qaID, tenantID).First(&qa).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "quick answer not found"})
		return
	}

	var ticket models.Ticket
	if err := db.Session(&gorm.Session{NewDB: true}).
		Preload("Contact").
		Where("id = ? AND \"tenantId\" = ?", input.TicketID, tenantID).
		First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	contact := ticket.Contact
	if contact.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}

	autoVars := map[string]string{
		"contact_name": ticket.Contact.Name,
		"ticket_id":    strconv.Itoa(ticket.ID),
		"agent_name":   tenantID.String(),
		"company_name": tenantID.String(),
	}
	for k, v := range input.Variables {
		autoVars[k] = v
	}

	content := utils.InterpolateVariables(qa.Content, autoVars)
	message := utils.InterpolateVariables(qa.Message, autoVars)

	var contentMap map[string]interface{}
	if content != "" && content != "{}" {
		_ = json.Unmarshal([]byte(content), &contentMap)
	}

	to := contactJID(contact)
	qaType := qa.Type
	if qaType == "" {
		qaType = "text"
	}

	sessionID := ticket.WhatsappID
	msgID := newWAMessageID()
	commandType, payload := flow.BuildQuickAnswerCommand(qaType, message, contentMap, sessionID, msgID, to)

	var whatsapp models.Whatsapp
	if err := db.Session(&gorm.Session{NewDB: true}).
		Where("id = ? AND \"tenantId\" = ?", ticket.WhatsappID, tenantID).First(&whatsapp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	if whatsapp.EngineType != "" && whatsapp.EngineType != "whatsmeow" {
		if qac.engines == nil {
			utils.RespondWithInternalError(c, fmt.Errorf("no engine resolver configured"), "SendQuickAnswer")
			return
		}
		engine, err := qac.engines.EngineFor(whatsapp)
		if err != nil {
			utils.RespondWithInternalError(c, err, "SendQuickAnswer")
			return
		}
		switch commandType {
		case "message.send.text":
			body, _ := payload["body"].(string)
			_, err = engine.SendText(c.Request.Context(), whatsapp, to, msgID, body)
		case "message.send.media":
			mediaURL, _ := payload["mediaUrl"].(string)
			mediaType, _ := payload["mediaType"].(string)
			mimeType, _ := payload["mimeType"].(string)
			_, err = engine.SendMedia(c.Request.Context(), whatsapp, to, msgID, mediaType, mediaURL, mimeType)
		default:
			richEngine, ok := engine.(domain.RichMessageEngine)
			if !ok {
				err = fmt.Errorf("resposta rápida do tipo %q não suportada no engine %q", qaType, whatsapp.EngineType)
				break
			}
			req := flow.BuildRichMessageRequest(qaType, message, contentMap)
			_, err = richEngine.SendInteractive(c.Request.Context(), whatsapp, to, msgID, req)
		}
		if err != nil {
			utils.RespondWithInternalError(c, err, "SendQuickAnswer")
			return
		}
	} else {
		command := map[string]interface{}{
			"type":    commandType,
			"payload": payload,
		}

		routingKey := wbot.Command(tenantID.String(), strconv.Itoa(ticket.WhatsappID), commandType)
		if err := qac.rabbit.PublishCommand(routingKey, command); err != nil {
			utils.RespondWithInternalError(c, err, "SendQuickAnswer")
			return
		}
	}

	// Persist outgoing message so it appears in the UI and ack events can find it.
	writeDB := qac.db.Session(&gorm.Session{NewDB: true})
	contactID := ticket.ContactID
	now := time.Now()

	msgBody := message
	msgMediaURL := ""
	msgMediaType := ""
	if qaType == "media" {
		if mu, ok := contentMap["url"].(string); ok {
			msgMediaURL = mu
		}
		if mt, ok := contentMap["mediaType"].(string); ok {
			msgMediaType = mt
		}
	}

	msgData := map[string]interface{}{
		"quickAnswerId": qa.ID,
		"qaType":        qaType,
	}
	if len(contentMap) > 0 {
		msgData["content"] = contentMap
	}
	msgDataJSON, _ := json.Marshal(msgData)

	outgoing := models.Message{
		ID:        msgID,
		Body:      msgBody,
		Ack:       0,
		MediaType: msgMediaType,
		MediaUrl:  msgMediaURL,
		TicketID:  input.TicketID,
		FromMe:    true,
		ContactID: &contactID,
		TenantID:  tenantID,
		Reactions: "[]",
		DataJson:  string(msgDataJSON),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := writeDB.Create(&outgoing).Error; err != nil {
		utils.RespondWithInternalError(c, err, "SendQuickAnswer-CreateMessage")
		return
	}

	lastMessage := msgBody
	if lastMessage == "" && msgMediaType != "" {
		lastMessage = "[" + msgMediaType + "]"
	}

	writeDB.Model(&models.Ticket{}).
		Where("id = ? AND \"tenantId\" = ?", input.TicketID, tenantID).
		Updates(map[string]interface{}{
			"lastMessage": lastMessage,
			"updatedAt":   now,
		})

	ticketRoom := "chat:" + strconv.Itoa(input.TicketID)
	msgPayload := map[string]interface{}{
		"action":  "create",
		"message": outgoing,
	}
	qac.broadcast.EmitToRoom("/", ticketRoom, "appMessage", msgPayload)
	qac.broadcast.EmitToTenantRoom(tenantID.String(), "appMessage", msgPayload)
	qac.broadcast.EmitToTenantRoom(tenantID.String(), "ticket", map[string]interface{}{
		"action": "update",
		"ticket": map[string]interface{}{
			"id":          input.TicketID,
			"lastMessage": lastMessage,
			"updatedAt":   now,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "sent"})
}

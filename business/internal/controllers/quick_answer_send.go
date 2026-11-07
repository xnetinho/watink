package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/flow"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
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

	sessionID := ticket.ChannelConnectionID
	msgID := newWAMessageID()
	commandType, payload := flow.BuildQuickAnswerCommand(qaType, message, contentMap, sessionID, msgID, to)

	conn, ch, found, cerr := channelOf(db, tenantID, ticket)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}
	sender, can := ch.(domain.RichSender)
	if cerr != nil || !can {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "quick_answer_not_supported"})
		return
	}
	if err := sender.SendRich(c.Request.Context(), &conn, commandType, payload); err != nil {
		utils.RespondWithInternalError(c, err, "SendQuickAnswer")
		return
	}

	// Persist outgoing message so it appears in the UI and ack events can find it.
	// db é o handle escopado da empresa do request (RLS); qac.db é o pool global e o banco recusa a gravação sem escopo.
	writeDB := db.Session(&gorm.Session{NewDB: true})
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
	// Enquete: o cartão da tela e o tratamento de voto leem as opções em rich (a mesma forma da enquete recebida). Sem isso o
	// voto do contato era ignorado e o cartão não mostrava a contagem.
	if qaType == "poll" {
		msgMediaType = "poll"
		var options []string
		if raw, ok := contentMap["options"].([]interface{}); ok {
			for _, o := range raw {
				if str, ok := o.(string); ok {
					options = append(options, str)
				}
			}
		}
		selectable := 1
		if ms, ok := contentMap["maxSelections"].(float64); ok && ms >= 1 {
			selectable = int(ms)
		}
		msgData["rich"] = map[string]interface{}{"options": options, "selectable": selectable, "question": contentMap["question"]}
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

	lastMessage, lastKind := domain.LastMessagePreview(msgMediaType, msgBody, "")

	writeDB.Model(&models.Ticket{}).
		Where("id = ? AND \"tenantId\" = ?", input.TicketID, tenantID).
		Updates(map[string]interface{}{
			"lastMessage":     lastMessage,
			"lastMessageType": lastKind,
			"updatedAt":       now,
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
			"id":              input.TicketID,
			"lastMessage":     lastMessage,
			"lastMessageType": lastKind,
			"updatedAt":       now,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "sent"})
}

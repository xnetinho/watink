package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// EditWindow é quanto tempo depois do envio o WhatsApp aceita editar (whatsmeow.EditWindow). Passado o prazo o servidor
// recusa a edição; barrar aqui evita enviar um comando que não vai funcionar.
const EditWindow = 20 * time.Minute

// editableMedia: mídias cuja LEGENDA o WhatsApp deixa editar. O arquivo em si nunca é editável.
var editableMedia = map[string]bool{"": true, "chat": true, "image": true, "video": true, "document": true}

// EditMessage edita o texto (ou a legenda) de uma mensagem que o operador enviou pelo WhatsApp.
//
// Regras: só mensagem nossa, não apagada, de texto ou mídia com legenda, no máximo 20 min depois do envio, e já aceita
// pelo servidor (ack >= 1: antes disso a mensagem nem saiu e não há o que editar). A edição entra na cadeia de versões
// (a versão 1 é o texto original) e a tela é avisada em tempo real.
//
// @Summary      Editar mensagem enviada
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        messageId  path  string  true  "ID da mensagem"
// @Success      200        {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /message/{messageId}/edit [post]
func (mc *MessageController) EditMessage(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Messages")
	if !ok {
		return
	}
	var input struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty_body"})
		return
	}
	if _, err := utils.ValidateStringField(body, "body", 65535); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var src models.Message
	if err := db.Session(&gorm.Session{NewDB: true}).Preload("Ticket").Preload("Ticket.Contact").
		Where(`id = ? AND "tenantId" = ?`, c.Param("messageId"), tenantID).First(&src).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	if code, status := editRefusal(src, time.Now()); code != "" {
		c.JSON(status, gin.H{"error": code})
		return
	}
	// Só o WhatsApp edita por este comando. Canais da tabela ChannelConnections (Telegram etc.) têm API própria.
	if src.Ticket.ChannelConnectionID != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "edit_not_supported"})
		return
	}
	if src.Ticket.Contact.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}
	if body == strings.TrimSpace(src.Body) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "edit_unchanged"})
		return
	}

	command := map[string]interface{}{
		"type": wbot.CommandMessageEdit,
		"payload": map[string]interface{}{
			"sessionId":   src.Ticket.WhatsappID,
			"messageId":   newWAMessageID(),
			"to":          contactJID(src.Ticket.Contact),
			"targetMsgId": src.ID,
			"body":        body,
			"mediaType":   src.MediaType,
		},
	}
	routingKey := wbot.Command(tenantID.String(), strconv.Itoa(src.Ticket.WhatsappID), wbot.CommandMessageEdit)
	if err := mc.rabbit.PublishCommand(routingKey, command); err != nil {
		utils.RespondWithInternalError(c, err, "EditMessage")
		return
	}

	// A edição que NÓS enviamos não volta pelo barramento: grava a versão e avisa a tela aqui, como na reação.
	res, err := services.RecordEdit(c.Request.Context(), db.Session(&gorm.Session{NewDB: true}), services.RecordEditInput{
		TenantID: tenantID, MessageID: src.ID, Body: body, EditedAt: time.Now(), EditedBy: models.EditedByMe,
	})
	if err != nil {
		utils.RespondWithInternalError(c, err, "EditMessage")
		return
	}
	mc.broadcast.EmitToRoom("/", "chat:"+strconv.Itoa(src.TicketID), "appMessage", gin.H{"action": "update", "message": res.Message})
	c.JSON(http.StatusOK, gin.H{"message": "Message edited", "editCount": res.Message.EditCount})
}

// editRefusal diz por que a mensagem não pode ser editada agora ("" = pode). Separada para ser testada sem HTTP.
func editRefusal(m models.Message, now time.Time) (code string, status int) {
	switch {
	case !m.FromMe || m.IsDeleted:
		return "message_not_editable", http.StatusUnprocessableEntity
	case !editableMedia[m.MediaType]:
		return "message_not_editable", http.StatusUnprocessableEntity
	case m.Ack < 1:
		// Ack 0: o engine ainda não confirmou o envio. Ack 5: falhou e não chegou; para esse caso existe o reenvio.
		return "message_not_sent", http.StatusConflict
	case m.Ack == ackFailedMessage:
		return "message_not_sent", http.StatusConflict
	case now.Sub(m.CreatedAt) > EditWindow:
		return "edit_window_expired", http.StatusUnprocessableEntity
	}
	return "", 0
}

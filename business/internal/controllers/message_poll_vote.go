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

// pollOptions lê as opções e o limite de seleção gravados em dataJson.rich ao receber a enquete.
func pollOptions(dataJSON string) (options []string, selectable int) {
	var d struct {
		Rich struct {
			Options    []string `json:"options"`
			Selectable int      `json:"selectable"`
		} `json:"rich"`
	}
	_ = json.Unmarshal([]byte(dataJSON), &d)
	return d.Rich.Options, d.Rich.Selectable
}

// validatePollVote confere o voto contra a enquete: só opções que existem, sem repetir, e no máximo o limite.
// Lista vazia é válida (remove o voto).
func validatePollVote(poll, chosen []string, selectable int) string {
	valid := make(map[string]bool, len(poll))
	for _, o := range poll {
		valid[o] = true
	}
	seen := map[string]bool{}
	for _, c := range chosen {
		if !valid[c] {
			return "opção inexistente na enquete"
		}
		if seen[c] {
			return "opção repetida"
		}
		seen[c] = true
	}
	if selectable > 0 && len(chosen) > selectable {
		return "mais opções do que a enquete permite"
	}
	return ""
}

// VotePoll registra o voto do operador em uma enquete recebida no WhatsApp.
//
// @Summary      Votar em enquete
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        messageId  path  string  true  "ID da mensagem da enquete"
// @Success      200  {object}  map[string]string
// @Security     BearerAuth
// @Router       /message/{messageId}/vote [post]
func (mc *MessageController) VotePoll(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Messages")
	if !ok {
		return
	}
	var input struct {
		Options []string `json:"options"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}

	var target models.Message
	if err := db.Preload("Ticket").Preload("Ticket.Contact").
		Where("id = ? AND \"tenantId\" = ?", c.Param("messageId"), tenantID).First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	if target.MediaType != "poll" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "a mensagem não é uma enquete"})
		return
	}
	// Só o WhatsApp vota: o Telegram só entrega a enquete como texto aqui.
	if target.Ticket.ChannelConnectionID != nil || target.Ticket.WhatsappID == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "votar só é possível em enquetes do WhatsApp"})
		return
	}
	options, selectable := pollOptions(target.DataJson)
	if len(options) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "enquete sem opções estruturadas (recebida antes do suporte a voto)"})
		return
	}
	if msg := validatePollVote(options, input.Options, selectable); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	var wa models.Whatsapp
	if err := db.Session(&gorm.Session{NewDB: true}).
		Where("id = ? AND \"tenantId\" = ?", target.Ticket.WhatsappID, tenantID).First(&wa).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Connection not found"})
		return
	}

	// O whatsmeow guardou o segredo da enquete com o JID com que ela chegou (LID em conversa 1:1; participante em grupo).
	// O participant gravado pode ser o telefone resolvido, que não acha o segredo: em 1:1 vale o JID do chat.
	chat := contactJID(target.Ticket.Contact)
	sender := chat
	if target.Ticket.Contact.IsGroup && target.Participant != "" {
		sender = target.Participant
	}
	command := map[string]interface{}{
		"type": "message.vote.poll",
		"payload": map[string]interface{}{
			"sessionId":  target.Ticket.WhatsappID,
			"chat":       chat,
			"pollId":     target.ID,
			"pollSender": sender,
			"pollFromMe": target.FromMe,
			"options":    input.Options,
		},
	}
	routingKey := wbot.Command(tenantID.String(), strconv.Itoa(target.Ticket.WhatsappID), wbot.CommandMessagePollVote)
	if err := mc.rabbit.PublishCommand(routingKey, command); err != nil {
		utils.RespondWithInternalError(c, err, "VotePoll")
		return
	}

	// Guarda o voto do operador no dataJson para a tela mostrar o que foi escolhido.
	d := map[string]interface{}{}
	_ = json.Unmarshal([]byte(target.DataJson), &d)
	d["myVote"] = input.Options
	if b, err := json.Marshal(d); err == nil {
		writeDB := db.Session(&gorm.Session{NewDB: true})
		writeDB.Model(&models.Message{}).Where("id = ? AND \"tenantId\" = ?", target.ID, tenantID).Update("dataJson", string(b))
		target.DataJson = string(b)
	}
	mc.broadcast.EmitToRoom("/", "chat:"+strconv.Itoa(target.TicketID), "appMessage", map[string]interface{}{"action": "update", "message": target})

	c.JSON(http.StatusOK, gin.H{"message": "Vote sent"})
}

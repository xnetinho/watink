package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxForwardTargets = 20

// forwardableMedia: tipos cujo conteúdo o envio sabe reproduzir. Enquete, localização, contato, PIX, chamada e
// visualização única são cartões do canal de origem e não têm envio equivalente.
var forwardableMedia = map[string]bool{"": true, "chat": true, "image": true, "video": true, "audio": true, "document": true, "sticker": true}

type forwardResult struct {
	TicketID int    `json:"ticketId"`
	OK       bool   `json:"ok"`
	Code     string `json:"code,omitempty"`
}

// ForwardMessage encaminha uma mensagem para um ou vários tickets do MESMO canal.
//
//	mode "forward": encaminhamento de verdade (selo "Encaminhada"; o canal precisa suportar)
//	mode "copy":    reenvio como mensagem nova, sem selo
//
// Entre canais diferentes é recusado: o conteúdo de um canal não vira mensagem de outro.
//
// @Summary      Encaminhar mensagem
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        messageId  path  string  true  "ID da mensagem"
// @Success      200        {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /message/{messageId}/forward [post]
func (mc *MessageController) ForwardMessage(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Messages")
	if !ok {
		return
	}

	var input struct {
		TicketIDs []int  `json:"ticketIds"`
		Mode      string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.TicketIDs) == 0 || len(input.TicketIDs) > maxForwardTargets {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_targets"})
		return
	}
	if input.Mode != "forward" && input.Mode != "copy" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_mode"})
		return
	}

	var src models.Message
	if err := db.Session(&gorm.Session{NewDB: true}).Preload("Ticket").
		Where(`id = ? AND "tenantId" = ?`, c.Param("messageId"), tenantID).First(&src).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	if src.IsDeleted || !forwardableMedia[src.MediaType] {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "message_not_forwardable"})
		return
	}
	if src.MediaType != "" && src.MediaType != "chat" && src.MediaUrl == "" {
		// Mídia ainda não baixada: não há o que reenviar.
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "media_unavailable"})
		return
	}

	srcKind, _ := resolveTicketChannelAndCaps(db, &src.Ticket)
	srcConn := connectionKey(&src.Ticket)

	results := make([]forwardResult, 0, len(input.TicketIDs))
	seen := map[int]bool{}
	for _, tid := range input.TicketIDs {
		if seen[tid] {
			continue
		}
		seen[tid] = true
		results = append(results, mc.forwardOne(c, db, tenantID, src, srcKind, srcConn, tid, input.Mode == "copy"))
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

// connectionKey identifica a conexão do ticket entre as duas tabelas (ChannelConnections e Whatsapps).
func connectionKey(t *models.Ticket) string {
	if t.ChannelConnectionID != nil {
		return "c" + strconv.Itoa(*t.ChannelConnectionID)
	}
	return "w" + strconv.Itoa(t.WhatsappID)
}

func (mc *MessageController) forwardOne(c *gin.Context, db *gorm.DB, tenantID uuid.UUID, src models.Message, srcKind, srcConn string, targetID int, asNew bool) forwardResult {
	fail := func(code string) forwardResult { return forwardResult{TicketID: targetID, Code: code} }

	var target models.Ticket
	if err := db.Session(&gorm.Session{NewDB: true}).Preload("Contact").
		Where(`id = ? AND "tenantId" = ?`, targetID, tenantID).First(&target).Error; err != nil || target.Contact.ID == 0 {
		return fail("ticket_not_found")
	}
	if kind, _ := resolveTicketChannelAndCaps(db, &target); kind != srcKind {
		return fail("different_channel")
	}
	if p := resolveTicketReplyPolicy(db, &target); !p.CanReply {
		return fail("reply_not_allowed")
	}

	score := dataFieldInt(src.DataJson, "forwardScore")
	data := map[string]interface{}{"mimetype": dataFieldString(src.DataJson, "mimetype")}
	if !asNew {
		data["forwarded"], data["forwardScore"] = true, score+1
	}
	dj, _ := json.Marshal(data)

	res, _, f := mc.deliver(c.Request.Context(), db, tenantID, target, outboundInput{
		Body: src.Body, MediaType: src.MediaType, MediaURL: src.MediaUrl,
		MimeType: dataFieldString(src.DataJson, "mimetype"),
		DataJSON: string(dj),
		Forward: &forwardInput{
			Source: src, SourceChatKey: src.Ticket.ConversationKey, AsNew: asNew,
			SameConnection: srcConn == connectionKey(&target), SourceForwarded: score,
		},
	})
	if f != nil {
		if f.msg == "forward_not_supported" {
			return fail("forward_not_supported")
		}
		return fail("send_failed")
	}
	mc.announce(db, tenantID, res)
	return forwardResult{TicketID: targetID, OK: true}
}

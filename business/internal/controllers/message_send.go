package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	wbot "github.com/alltomatos/watinkdev/contracts/wbot"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/mediastore"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// newWAMessageID generates a WhatsApp-compatible outgoing message ID
// (uppercase hex, "3EB0" prefix — the de-facto format for bot-sent messages).
func newWAMessageID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("3EB0%016X", time.Now().UnixNano())
	}
	return "3EB0" + strings.ToUpper(hex.EncodeToString(b))
}

// contactJID builds the destination JID for a contact. Groups use the "@g.us"
// server. LID contacts use the full "@lid" JID stored in the Lid field.
// Regular users are sent bare (the engine appends "@s.whatsapp.net").
// quotedPreviewText é o texto que o celular do contato mostra na caixa de citação. Mídia sem legenda vira o
// rótulo do tipo, como o próprio WhatsApp faz.
func quotedPreviewText(q models.Message) string {
	if q.Body != "" && q.MediaType != "call" && q.MediaType != "view_once" {
		return q.Body
	}
	switch q.MediaType {
	case "image":
		return "📷 Foto"
	case "video":
		return "🎥 Vídeo"
	case "audio":
		return "🎵 Áudio"
	case "document":
		return "📄 Documento"
	case "sticker":
		return "Figurinha"
	}
	return q.Body
}

func contactJID(contact models.Contact) string {
	return utils.ContactJID(contact.Number, contact.Lid, contact.IsGroup)
}

// SendMessage envia uma mensagem via engine WhatsApp.
// Aceita JSON (texto) ou multipart/form-data (mídia com arquivo binário).
//
// @Summary      Enviar mensagem
// @Description  Envia mensagem de texto ou mídia via engine WhatsApp
// @Tags         messages
// @Accept       json
// @Accept       mpfd
// @Produce      json
// @Param        ticketId  path      int     true  "ID do ticket"
// @Param        body      body      object  false "Corpo JSON para texto"
// @Param        medias    formData  file    false "Arquivo de mídia (multipart)"
// @Param        fromMe    formData  string  false "true/false"
// @Success      200       {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /messages/{ticketId} [post]
func (mc *MessageController) SendMessage(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Messages")
	if !ok {
		return
	}

	ticketIDStr := c.Param("ticketId")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	// Preload the contact so we can resolve the destination JID. We must not reuse
	// the (already tenant-scoped) db for a second .First() — GORM would accumulate
	// the ticket's id filter into the contact query (id = ticketID AND id = contactID).
	var ticket models.Ticket
	if err := db.Preload("Contact").Where("id = ? AND \"tenantId\" = ?", ticketID, tenantID).First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	contact := ticket.Contact
	if contact.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	// Validação de regra de resposta por canal (D2 / ADR 0033)
	policy := resolveTicketReplyPolicy(db, &ticket)
	if !policy.CanReply {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "reply_not_allowed",
			"code":    string(policy.Reason),
			"message": policy.Message,
		})
		return
	}

	ct := c.ContentType()
	var body, mediaType, mediaURL, mimeType, quotedID string

	if ct == "application/json" || ct == "" {
		var input struct {
			Body      string `json:"body"`
			MediaType string `json:"mediaType"`
			MediaUrl  string `json:"mediaUrl"`
			QuotedMsg *struct {
				ID string `json:"id"`
			} `json:"quotedMsg"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			utils.RespondWithBindError(c, err)
			return
		}
		if _, err := utils.ValidateStringField(input.Body, "body", 65535); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := utils.ValidateStringField(input.MediaType, "mediaType", 50); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := utils.ValidateStringField(input.MediaUrl, "mediaUrl", 2048); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		body = input.Body
		mediaType = input.MediaType
		mediaURL = input.MediaUrl
		if input.QuotedMsg != nil {
			quotedID = input.QuotedMsg.ID
		}
	} else {
		// multipart/form-data: arquivo(s) de mídia
		body = c.PostForm("body")
		quotedID = c.PostForm("quotedMsgId")
		file, header, err := c.Request.FormFile("medias")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "medias file required for multipart"})
			return
		}
		defer func() { _ = file.Close() }()

		mimeType = sanitizeMimeType(header.Header.Get("Content-Type"))

		savedURL, err := mediastore.SaveMediaReader(file, mimeType)
		if err != nil {
			log.Printf("[SendMessage] media save failed (ticket %d)", ticketID)
			utils.RespondWithInternalError(c, err, "SendMessage.SaveMedia")
			return
		}

		mediaURL = savedURL
		mediaType = mimeTypeToMediaType(mimeType)
	}

	messageID := newWAMessageID()
	to := contactJID(contact)

	// Só cita uma mensagem que existe NESTE ticket e tenant: o id vem do cliente e o engine
	// o usaria como StanzaID. quotedParticipant é quem escreveu a original (obrigatório em
	// grupo; em chat 1:1 é o próprio contato, ou nós mesmos se citamos algo que enviamos).
	var quotedParticipant, quotedBody string
	quotedExternalID := quotedID // id da citada no canal; difere de quotedID fora do WhatsApp
	if quotedID != "" {
		var q models.Message
		if err := db.Session(&gorm.Session{NewDB: true}).
			Where(`id = ? AND "ticketId" = ? AND "tenantId" = ?`, quotedID, ticketID, tenantID).First(&q).Error; err != nil {
			quotedID = ""
		} else {
			quotedBody = quotedPreviewText(q)
			if q.ExternalID != "" {
				quotedExternalID = q.ExternalID
			}
			if !q.FromMe {
				quotedParticipant = q.Participant
				// 1:1: quem escreveu a citada é o próprio contato, no endereçamento do chat. O Participant gravado
				// pode ser o número (PN) enquanto o chat é por LID, e o celular não casa os dois.
				if quotedParticipant == "" || !contact.IsGroup {
					quotedParticipant = to
				}
			}
		}
	}

	// Se o ticket pertencer a uma ChannelConnection agnóstica registrada, despacha pelo canal (Tarefa 4.2 / ADR 0033)
	if ticket.ChannelConnectionID != nil {
		var conn models.ChannelConnection
		if err := db.Session(&gorm.Session{NewDB: true}).
			Where("id = ? AND \"tenantId\" = ?", *ticket.ChannelConnectionID, tenantID).
			First(&conn).Error; err == nil {
			if chAdapter, err := domain.DefaultChannelRegistry().Get(domain.ChannelKind(conn.Kind)); err == nil {
				outMsg := domain.OutboundMessage{
					ConversationKey: ticket.ConversationKey,
					ExternalID:      to,
					Body:            body,
					MediaType:       mediaType,
					MediaURL:        mediaURL,
					QuotedMessageID: quotedExternalID,
				}
				sendRes, err := chAdapter.Send(c.Request.Context(), &conn, outMsg)
				if err != nil {
					utils.RespondWithInternalError(c, err, "SendMessage")
					return
				}
				externalID := sendRes.ExternalMessageID
				if externalID != "" {
					messageID = domain.StorageMessageID(domain.ChannelKind(conn.Kind), conn.ID, ticket.ConversationKey, externalID)
				}
				msg := models.Message{
					ID:         messageID,
					ExternalID: externalID,
					Body:       body,
					Ack:        1,
					MediaType:  mediaType,
					MediaUrl:   mediaURL,
					TicketID:   ticket.ID,
					ContactID:  &contact.ID,
					FromMe:     true,
					TenantID:   tenantID,
				}
				if quotedID != "" {
					msg.QuotedMsgID = &quotedID
				}
				// Session(NewDB:true): o db carrega filtros acumulados das consultas acima e o GORM os usaria no INSERT.
				if err := db.Session(&gorm.Session{NewDB: true}).Create(&msg).Error; err != nil {
					// A mensagem já saiu para o cliente: loga e responde com o que foi enviado, sem fingir que falhou.
					log.Printf("[SendMessage] mensagem enviada mas não persistida (ticket %d): %v", ticket.ID, err)
				}
				c.JSON(http.StatusOK, msg)
				return
			}
		}
	}

	var whatsapp models.Whatsapp
	if err := db.Session(&gorm.Session{NewDB: true}).
		Where("id = ? AND \"tenantId\" = ?", ticket.WhatsappID, tenantID).First(&whatsapp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Connection not found"})
		return
	}

	if whatsapp.EngineType != "" && whatsapp.EngineType != "whatsmeow" {
		// Non-AMQP engines are dispatched via domain.WhatsAppEngine
		// instead of the raw wbot.<tenant>.<session>.<cmd> command below.
		if mc.engines == nil {
			utils.RespondWithInternalError(c, fmt.Errorf("no engine resolver configured"), "SendMessage")
			return
		}
		engine, err := mc.engines.EngineFor(whatsapp)
		if err != nil {
			utils.RespondWithInternalError(c, err, "SendMessage")
			return
		}
		// (a API não expõe citação; a resposta sai como mensagem comum.)
		if mediaURL != "" {
			_, err = engine.SendMedia(c.Request.Context(), whatsapp, to, messageID, mediaType, mediaURL, mimeType)
		} else {
			_, err = engine.SendText(c.Request.Context(), whatsapp, to, messageID, body)
		}
		if err != nil {
			utils.RespondWithInternalError(c, err, "SendMessage")
			return
		}
	} else {
		commandType := "message.send.text"
		if mediaURL != "" {
			commandType = "message.send.media"
		}

		command := map[string]interface{}{
			"type": commandType,
			"payload": map[string]interface{}{
				"sessionId": ticket.WhatsappID,
				"messageId": messageID,
				"to":        to,
				"ticketId":  ticketID,
				"body":      body,
				"mediaType": mediaType,
				"mediaUrl":  mediaURL,
				"mimeType":  mimeType,
			},
		}
		if quotedID != "" {
			cmdPayload := command["payload"].(map[string]interface{})
			cmdPayload["quotedMsgId"] = quotedID
			cmdPayload["quotedBody"] = quotedBody
			if quotedParticipant != "" {
				cmdPayload["quotedJid"] = quotedParticipant
			}
		}

		// The engine dispatches by routing-key segment (wbot.<tenant>.<session>.<cmd>),
		// so the command type MUST be encoded in the key — not just the body.
		routingKey := wbot.Command(tenantID.String(), strconv.Itoa(ticket.WhatsappID), commandType)
		if err := mc.rabbit.PublishCommand(routingKey, command); err != nil {
			utils.RespondWithInternalError(c, err, "SendMessage")
			return
		}
	}

	// Persist the outgoing message so it appears in the UI immediately and the
	// later ack event (event_listener_msg_ack) can find and update it.
	// Use a clean session for writes — the scoped `db` already carries the tenant
	// Where clause and a finished SELECT statement, which corrupts the generated
	// SQL when reused for Create/Updates. NewDB clears accumulated conditions
	// while keeping the same connection; tenant safety is preserved because the
	// inserted row and the update filter both set tenantId explicitly.
	writeDB := db.Session(&gorm.Session{NewDB: true})
	contactID := ticket.ContactID
	now := time.Now()
	outgoing := models.Message{
		ID:        messageID,
		Body:      body,
		Ack:       0,
		MediaType: mediaType,
		MediaUrl:  mediaURL,
		TicketID:  ticketID,
		FromMe:    true,
		ContactID: &contactID,
		TenantID:  tenantID,
		Reactions: "[]",
		DataJson:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if quotedID != "" {
		outgoing.QuotedMsgID = &quotedID
	}
	if err := writeDB.Create(&outgoing).Error; err != nil {
		log.Printf("[SendMessage] persist outgoing message failed (ticket %d): %v", ticketID, err)
	}
	// O evento SSE abaixo é o que o navegador exibe na hora; só a listagem anexava a citação, então a
	// resposta aparecia sem ela até recarregar o chat.
	if quotedID != "" {
		one := []models.Message{outgoing}
		attachQuotedMessages(db, tenantID, one)
		outgoing.QuotedMsg = one[0].QuotedMsg
	}

	lastMessage, lastKind := domain.LastMessagePreview(mediaType, body, "")
	writeDB.Model(&models.Ticket{}).
		Where("id = ? AND \"tenantId\" = ?", ticketID, tenantID).
		Updates(map[string]interface{}{"lastMessage": lastMessage, "lastMessageType": lastKind, "updatedAt": now})

	// Emit socket events so the frontend updates in real-time without a page refresh.
	// Pattern mirrors event_listener.go: appMessage to the ticket room + tenant room,
	// and ticket:update to the tenant room so the sidebar lastMessage refreshes.
	ticketRoom := "chat:" + strconv.Itoa(ticketID)
	msgPayload := map[string]interface{}{"action": "create", "message": outgoing}
	mc.broadcast.EmitToRoom("/", ticketRoom, "appMessage", msgPayload)
	mc.broadcast.EmitToTenantRoom(tenantID.String(), "appMessage", msgPayload)
	mc.broadcast.EmitToTenantRoom(tenantID.String(), "ticket", map[string]interface{}{
		"action": "update",
		"ticket": map[string]interface{}{
			"id":              ticketID,
			"lastMessage":     lastMessage,
			"lastMessageType": lastKind,
			"updatedAt":       now,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Message sent", "messageId": messageID})
}

// sanitizeMimeType strips newlines and control characters from a user-supplied
// Content-Type header to prevent log injection. Falls back to
// "application/octet-stream" when the value is empty after sanitization.
func sanitizeMimeType(raw string) string {
	safe := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r < 0x20 {
			return -1
		}
		return r
	}, raw)
	if safe == "" {
		return "application/octet-stream"
	}
	return safe
}

func mimeTypeToMediaType(mimeType string) string {
	switch {
	case len(mimeType) >= 5 && mimeType[:5] == "image":
		return "image"
	case len(mimeType) >= 5 && mimeType[:5] == "video":
		return "video"
	case len(mimeType) >= 5 && mimeType[:5] == "audio":
		return "audio"
	default:
		return "document"
	}
}

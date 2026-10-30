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

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/mediastore"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
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
	// voice: o áudio é uma GRAVAÇÃO (bolha de voz com onda), não um arquivo de áudio. Só vale para mediaType=audio.
	var voice bool
	// viewOnce: foto, vídeo ou áudio enviado para ser visto uma única vez (olho no painel de envio).
	var viewOnce bool
	// fileName é o nome ORIGINAL do arquivo anexado: o destinatário vê este nome, não o hash com que o arquivo é salvo.
	var fileName string

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
		voice = c.PostForm("audioKind") == "voice"
		viewOnce = c.PostForm("viewOnce") == "true"
		file, header, err := c.Request.FormFile("medias")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "medias file required for multipart"})
			return
		}
		defer func() { _ = file.Close() }()

		mimeType = sanitizeMimeType(header.Header.Get("Content-Type"))
		fileName = sanitizeFileName(header.Filename)

		savedURL, err := mediastore.SaveMediaReader(file, mimeType)
		if err != nil {
			log.Printf("[SendMessage] media save failed (ticket %d)", ticketID)
			utils.RespondWithInternalError(c, err, "SendMessage.SaveMedia")
			return
		}

		mediaURL = savedURL
		mediaType = mimeTypeToMediaType(mimeType)
	}

	res, viaChannel, fail := mc.deliver(c.Request.Context(), db, tenantID, ticket, outboundInput{
		Body: body, MediaType: mediaType, MediaURL: mediaURL, MimeType: mimeType, QuotedID: quotedID,
		Voice:    voice && mediaType == "audio",
		ViewOnce: viewOnce && canBeViewOnce(mediaType),
		FileName: fileName,
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

// canBeViewOnce diz se o tipo aceita visualização única: foto, vídeo e áudio. Documento e figurinha não têm esse modo no
// WhatsApp; o pedido é ignorado em vez de falhar (o arquivo sai como mídia comum).
func canBeViewOnce(mediaType string) bool {
	switch mediaType {
	case "image", "video", "audio":
		return true
	}
	return false
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

// sanitizeFileName reduz o nome enviado pelo cliente a um nome simples: sem diretórios, sem caracteres de controle, com
// tamanho limitado. Ele vira o título que o destinatário vê e é gravado na mensagem.
func sanitizeFileName(raw string) string {
	name := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, raw)
	name = strings.TrimSpace(name)
	if name == "." || name == ".." {
		return ""
	}
	if r := []rune(name); len(r) > 120 {
		name = string(r[len(r)-120:]) // mantém o fim: é onde fica a extensão
	}
	return name
}

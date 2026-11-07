package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// forwardInput descreve o encaminhamento de uma mensagem já existente para outro ticket do mesmo canal.
type forwardInput struct {
	Source          models.Message
	SourceChatKey   string // conversationKey do ticket de origem (chat_id no Telegram)
	AsNew           bool   // true: reenvia como mensagem nova, sem o selo "Encaminhada"
	SameConnection  bool   // origem e destino são a mesma conexão (única que o canal consegue encaminhar nativamente)
	SourceForwarded int    // forwardScore da original, para o contador do WhatsApp
}

// outboundInput é tudo o que o envio precisa para sair por qualquer canal, venha do compositor ou do encaminhamento.
type outboundInput struct {
	Body, MediaType, MediaURL, MimeType, QuotedID string
	DataJSON                                      string
	Voice                                         bool // áudio enviado como gravação (PTT)
	ViewOnce                                      bool // foto, vídeo ou áudio enviado como visualização única
	Sticker                                       bool // figurinha pedida no painel (o encaminhamento de figurinha não passa por aqui)
	FileName                                      string
	Forward                                       *forwardInput
}

// deliverFailure separa "erro interno" (500 com log) de "recusa com mensagem" (status explícito).
type deliverFailure struct {
	status int
	msg    string
	err    error
	where  string
}

func (f *deliverFailure) respond(c *gin.Context) {
	if f.status != 0 {
		c.JSON(f.status, gin.H{"error": f.msg})
		return
	}
	utils.RespondWithInternalError(c, f.err, f.where)
}

// deliver envia a mensagem pelo canal do ticket e a grava. viaChannel diz que saiu por um canal agnóstico (Telegram...),
// cujo corpo de resposta é a própria mensagem; no WhatsApp a gravação é otimista e o resto fica por conta de announce.
func (mc *MessageController) deliver(ctx context.Context, db *gorm.DB, tenantID uuid.UUID, ticket models.Ticket, in outboundInput) (res models.Message, viaChannel bool, fail *deliverFailure) {
	contact := ticket.Contact
	ticketID := ticket.ID
	body, mediaType, mediaURL, mimeType, quotedID := in.Body, in.MediaType, in.MediaURL, in.MimeType, in.QuotedID
	// O mimetype fica gravado na mensagem: sem ele, encaminhar uma mídia que o operador enviou a reenviaria sem tipo.
	dataJSON := in.DataJSON
	if dataJSON == "" {
		dataJSON = "{}"
		d := map[string]interface{}{}
		if mimeType != "" {
			d["mimetype"] = mimeType
		}
		if in.Voice {
			d["voice"] = true // gravação (PTT): o encaminhamento precisa continuar sendo voz
		}
		if in.FileName != "" {
			d["fileName"] = in.FileName // o encaminhamento reenvia com o mesmo nome
		}
		if in.ViewOnce {
			d["viewOnce"] = true // a tela mostra o selo "Visualização única" também na mensagem que ENVIAMOS
		}
		if len(d) > 0 {
			if b, err := json.Marshal(d); err == nil {
				dataJSON = string(b)
			}
		}
	}

	messageID := newWAMessageID()
	to := contactJID(contact)

	// Só cita uma mensagem que existe NESTE ticket e tenant: o id vem do cliente e o engine
	// o usaria como StanzaID. quotedParticipant é quem escreveu a original (obrigatório em
	// grupo; em chat 1:1 é o próprio contato, ou nós mesmos se citamos algo que enviamos).
	var quotedParticipant, quotedBody string
	var quotedMedia map[string]interface{}
	quotedExternalID := quotedID // id da citada no canal; difere de quotedID fora do WhatsApp
	if quotedID != "" {
		var q models.Message
		if err := db.Session(&gorm.Session{NewDB: true}).
			Where(`id = ? AND "ticketId" = ? AND "tenantId" = ?`, quotedID, ticketID, tenantID).First(&q).Error; err != nil {
			quotedID = ""
		} else {
			quotedBody = quotedPreviewText(q)
			quotedMedia = quotedMediaOf(q)
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

	// Um só caminho: a conexão do ticket (ChannelConnections) decide o canal e o adaptador envia.
	connID := ticket.ChannelConnectionID
	var conn models.ChannelConnection
	if err := db.Session(&gorm.Session{NewDB: true}).
		Where("id = ? AND \"tenantId\" = ?", connID, tenantID).First(&conn).Error; err != nil {
		return res, false, &deliverFailure{status: http.StatusNotFound, msg: "Connection not found"}
	}
	chAdapter, err := domain.DefaultChannelRegistry().Get(domain.ChannelKind(conn.Kind))
	if err != nil {
		return res, false, &deliverFailure{err: err, where: "SendMessage"}
	}
	isWhatsApp := conn.Kind == string(domain.ChannelKindWhatsApp)

	// Opções que só o WhatsApp tem: pedir em outro canal e mandar como mídia comum enganaria o operador (o selo da tela diria
	// o contrário do que saiu), então recusa com mensagem clara.
	if in.ViewOnce && !isWhatsApp {
		return res, false, &deliverFailure{status: http.StatusUnprocessableEntity, msg: "view_once_not_supported"}
	}
	if in.Sticker && !isWhatsApp {
		return res, false, &deliverFailure{status: http.StatusUnprocessableEntity, msg: "sticker_not_supported"}
	}

	outMsg := domain.OutboundMessage{
		ConversationKey: ticket.ConversationKey,
		ExternalID:      to,
		Body:            body,
		MediaType:       mediaType,
		MediaURL:        mediaURL,
		Voice:           in.Voice,
		ViewOnce:        in.ViewOnce,
		Sticker:         in.Sticker,
		FileName:        in.FileName,
		StorageID:       messageID,
		Metadata:        map[string]string{},
	}
	if mimeType != "" {
		outMsg.Metadata["mimeType"] = mimeType
	}
	// Só cita o que foi validado acima (existe neste ticket e tenant): quotedID zera quando a citada é inválida.
	if quotedID != "" {
		outMsg.QuotedMessageID = quotedExternalID
	}
	if quotedID != "" && isWhatsApp {
		outMsg.Quoted = &domain.QuotedRef{Participant: quotedParticipant, Body: quotedBody, Media: quotedMedia}
	}
	if in.Forward != nil {
		// Encaminhar com selo exige que o canal suporte E que origem e destino sejam a MESMA conexão (o Telegram só
		// encaminha o que o próprio bot enxerga). A regra vale antes do adaptador para qualquer canal.
		if !in.Forward.AsNew && (!chAdapter.Capabilities().Forward || !in.Forward.SameConnection) {
			return res, false, &deliverFailure{status: http.StatusUnprocessableEntity, msg: "forward_not_supported"}
		}
		outMsg.Forward = &domain.ForwardRef{
			FromConversationKey: in.Forward.SourceChatKey,
			ExternalMessageID:   in.Forward.Source.ExternalID,
			AsNew:               in.Forward.AsNew,
			SameConnection:      in.Forward.SameConnection,
			Score:               in.Forward.SourceForwarded,
		}
	}
	sendRes, err := chAdapter.Send(ctx, &conn, outMsg)
	if errors.Is(err, domain.ErrForwardUnsupported) {
		return res, false, &deliverFailure{status: http.StatusUnprocessableEntity, msg: "forward_not_supported"}
	}
	if err != nil {
		return res, false, &deliverFailure{err: err, where: "SendMessage"}
	}

	// Escreve sempre com sessão limpa: o db escopado carrega o WHERE de tenant e um SELECT já terminado, o que corrompe o
	// SQL do INSERT. O tenantId é gravado explicitamente na própria linha.
	writeDB := db.Session(&gorm.Session{NewDB: true})
	contactID := ticket.ContactID
	now := time.Now()
	if isWhatsApp {
		// Gravação otimista: a mensagem aparece na hora e o ack do engine (event_listener_msg_ack) a atualiza depois.
		outgoing := models.Message{
			ID: messageID, Body: body, Ack: 0, MediaType: mediaType, MediaUrl: mediaURL, TicketID: ticketID, FromMe: true,
			ContactID: &contactID, TenantID: tenantID, Reactions: "[]", DataJson: dataJSON, CreatedAt: now, UpdatedAt: now,
		}
		if quotedID != "" {
			outgoing.QuotedMsgID = &quotedID
		}
		if err := writeDB.Create(&outgoing).Error; err != nil {
			log.Printf("[SendMessage] persist outgoing message failed (ticket %d): %v", ticketID, err)
		}
		return outgoing, false, nil
	}

	externalID := sendRes.ExternalMessageID
	if externalID != "" {
		messageID = domain.StorageMessageID(domain.ChannelKind(conn.Kind), conn.ID, ticket.ConversationKey, externalID)
	}
	msg := models.Message{
		ID: messageID, ExternalID: externalID, Body: body, Ack: 1, MediaType: mediaType, MediaUrl: mediaURL, TicketID: ticket.ID,
		ContactID: &contact.ID, FromMe: true, TenantID: tenantID, DataJson: dataJSON,
	}
	if quotedID != "" {
		msg.QuotedMsgID = &quotedID
	}
	if err := writeDB.Create(&msg).Error; err != nil {
		// A mensagem já saiu para o cliente: loga e responde com o que foi enviado, sem fingir que falhou.
		log.Printf("[SendMessage] mensagem enviada mas não persistida (ticket %d): %v", ticket.ID, err)
	}
	return msg, true, nil
}

// announce atualiza a prévia do ticket e avisa a tela em tempo real. O evento é o que o navegador exibe na hora; só a
// listagem anexava a citação, então a resposta aparecia sem ela até recarregar o chat.
func (mc *MessageController) announce(db *gorm.DB, tenantID uuid.UUID, outgoing models.Message) {
	writeDB := db.Session(&gorm.Session{NewDB: true})
	ticketID := outgoing.TicketID
	now := time.Now()
	if outgoing.QuotedMsgID != nil {
		one := []models.Message{outgoing}
		attachQuotedMessages(db, tenantID, one)
		outgoing.QuotedMsg = one[0].QuotedMsg
	}

	lastMessage, lastKind := domain.LastMessagePreview(outgoing.MediaType, outgoing.Body, "")
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
}

// dataFieldString lê um campo de texto do dataJson de uma mensagem (mimetype, fileName...).
func dataFieldString(dataJSON, key string) string {
	var d map[string]interface{}
	if json.Unmarshal([]byte(dataJSON), &d) != nil {
		return ""
	}
	s, _ := d[key].(string)
	return s
}

// dataFieldInt lê um campo numérico do dataJson (forwardScore).
func dataFieldInt(dataJSON, key string) int {
	var d map[string]interface{}
	if json.Unmarshal([]byte(dataJSON), &d) != nil {
		return 0
	}
	f, _ := d[key].(float64)
	return int(f)
}

// dataFieldBool lê um campo booleano do dataJson (voice).
func dataFieldBool(dataJSON, key string) bool {
	var d map[string]interface{}
	if json.Unmarshal([]byte(dataJSON), &d) != nil {
		return false
	}
	b, _ := d[key].(bool)
	return b
}

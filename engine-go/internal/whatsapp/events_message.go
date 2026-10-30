package whatsapp

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/encoding/protojson"
)

func (s *WhatsAppService) handleMessageEvent(client *whatsmeow.Client, id int, tenantID string, v *events.Message) {
	if v.Message == nil {
		return
	}

	if viewOnceInspectEnabled() {
		logViewOnceInspect("msg", inspectViewOnceMessage(v))
		logViewOnceInspect("ctx", inspectViewOnceContext(v.Message))
	}

	// O celular principal citou uma visualização única: a citação traz a mídia completa. Baixa e entrega ao business sobre o
	// aviso, como mídia comum. Em goroutine: o download demora e o laço de eventos é sequencial para a sessão inteira.
	if media, target, kind := quotedViewOnceMedia(v.Message); media != nil {
		go s.recoverViewOnce(context.Background(), client, tenantID, id, media, target, kind)
	}

	if mediaInspectEnabled() {
		if d := inspectMedia(v.Message); d != "" {
			log.Printf("[media.inspect] %s de=%s: %s", v.Info.ID, v.Info.Sender.String(), d)
		}
		if d := inspectQuote(v.Message); d != "" {
			log.Printf("[quote.inspect] %s de=%s: %s", v.Info.ID, v.Info.Sender.String(), d)
		}
	}

	// WhatsApp Status/Stories ("status@broadcast") are not conversations and must
	// not create tickets. They also frequently carry media whose synchronous
	// download here would stall the serial event loop for the whole session.
	if v.Info.Chat.String() == "status@broadcast" {
		return
	}

	// Poll votes are encrypted at WhatsApp level. Decrypt before publishing; the business
	// persists the event only when the originating poll opted into result capture.
	if v.Message.GetPollUpdateMessage() != nil {
		s.handlePollVoteEvent(client, id, tenantID, v)
		return
	}

	// Edição do contato: o aparelho manda um SecretEncryptedMessage MESSAGE_EDIT (texto novo cifrado com o segredo da
	// mensagem original), não o ProtocolMessage que o BuildEdit usa para enviar. Confirmado com edições reais no celular.
	if enc := v.Message.GetSecretEncryptedMessage(); enc != nil && enc.GetSecretEncType() == waE2E.SecretEncryptedMessage_MESSAGE_EDIT {
		s.handleSecretEdit(client, id, tenantID, v)
		return
	}

	if protocolMsg := v.Message.GetProtocolMessage(); protocolMsg != nil {
		if protocolMsg.GetType() == waProto.ProtocolMessage_MESSAGE_EDIT && protocolMsg.GetKey() != nil {
			// Edição: o whatsmeow desembrulha o EditedMessage (v.IsEdit) e Info.ID NÃO é o da mensagem original; o id dela
			// está na chave do ProtocolMessage. Antes a edição era descartada aqui sem aviso.
			s.publishEvent(tenantID, id, "message.edited", editedPayload(id, v, protocolMsg.GetKey().GetID(), protocolMsg))
			return
		}
		if protocolMsg.GetType() == waProto.ProtocolMessage_REVOKE && protocolMsg.GetKey() != nil {
			s.publishEvent(tenantID, id, "message.revoke", map[string]interface{}{
				"sessionId": fmt.Sprintf("%d", id),
				"messageId": protocolMsg.GetKey().GetID(),
				"fromJid":   v.Info.Sender.String(),
				"fromMe":    v.Info.IsFromMe,
			})
		}
		return
	}

	if reactionMsg := v.Message.GetReactionMessage(); reactionMsg != nil && reactionMsg.GetKey() != nil {
		s.publishEvent(tenantID, id, "message.reaction", map[string]interface{}{
			"sessionId": fmt.Sprintf("%d", id),
			"messageId": reactionMsg.GetKey().GetID(),
			"jid":       v.Info.Sender.String(),
			"reaction":  reactionMsg.GetText(),
			"fromMe":    v.Info.IsFromMe,
		})
		return
	}

	content := extractMessageContent(v.Message)
	// Skip system / unsupported / empty messages (no text and no media) so they
	// don't show up in the chat as blank bubbles rendering the raw number.
	if content.msgType == "chat" && content.body == "" && content.protoB64 == "" {
		return
	}
	if content.msgType == "unsupported" {
		log.Printf("[Session-%d] mensagem de tipo não suportado (%s) id=%s: avisando o operador", id, content.body, v.Info.ID)
		logRawMessage(id, v.Info.ID, v.Message)
	}

	isGroup := v.Info.IsGroup || v.Info.Chat.Server == types.GroupServer
	mentionedJIDs := extractMentionedJIDs(v.Message)
	quotedMsgID := extractQuotedStanzaID(v.Message)
	chatJID := v.Info.Chat.String()
	// Strip the device suffix (sender:NN) from the participant.
	senderJID := v.Info.Sender.ToNonAD().String()
	if chatJID == "" {
		chatJID = senderJID
	}

	// Resolve LID → phone number (also device-stripped).
	resolvedSender := senderJID
	if v.Info.Sender.Server == types.HiddenUserServer {
		if pn, err := client.Store.LIDs.GetPNForLID(context.Background(), v.Info.Sender); err == nil && !pn.IsEmpty() {
			resolvedSender = pn.ToNonAD().String()
		}
	}

	// Group subject becomes the contact name (cached); avoids groups looking like
	// individuals named after whoever messaged. groupSubject also populates the
	// groupMetaMap cache used below for community/sub-group detection.
	groupName := ""
	isCommunity := false
	isSubGroup := false
	if isGroup {
		groupName = s.groupSubject(client, v.Info.Chat)
		meta := s.cachedGroupMeta(v.Info.Chat)
		isCommunity = meta.isCommunity
		isSubGroup = meta.isSubGroup
	}

	// Profile picture for the contact record (group's own photo or individual sender's photo).
	// LID senders (@lid, increasingly common) previously never got a picture
	// fetched — GetProfilePictureInfo needs a real JID, so resolve LID→PN
	// first (same resolution already done above for resolvedSender) instead
	// of only accepting DefaultUserServer.
	profilePic := ""
	senderPic := ""
	if isGroup {
		profilePic = s.getCachedPic(client, id, tenantID, v.Info.Chat.ToNonAD())
		// Individual participant photo — shown in message bubble avatars.
		if !v.Info.IsFromMe {
			if picJID, ok := s.resolvePicJID(client, v.Info.Sender); ok {
				senderPic = s.getCachedPic(client, id, tenantID, picJID)
			}
		}
	} else if !v.Info.IsFromMe {
		if picJID, ok := s.resolvePicJID(client, v.Info.Sender); ok {
			profilePic = s.getCachedPic(client, id, tenantID, picJID)
		}
	}

	// For outbound individual messages (fromMe=true, non-group), PushName is the
	// connection owner's name — publishing it would rename the recipient contact to
	// the agent's name. Send empty string so FindOrCreate never overwrites the
	// recipient's existing name with the owner's.
	pushName := v.Info.PushName
	if v.Info.IsFromMe && !isGroup {
		pushName = ""
	}

	forwarded, fwdScore := forwardedOf(v.Message)
	s.publishEvent(tenantID, id, "message.received", map[string]interface{}{
		"sessionId": fmt.Sprintf("%d", id),
		"message": map[string]interface{}{
			"id":            v.Info.ID,
			"from":          chatJID,
			"body":          content.body,
			"type":          content.msgType,
			"fromMe":        v.Info.IsFromMe,
			"timestamp":     v.Info.Timestamp.Unix(),
			"pushName":      pushName,
			"groupName":     groupName,
			"profilePicUrl": profilePic,
			"senderPicUrl":  senderPic,
			"isLid":         v.Info.Sender.Server == types.HiddenUserServer,
			"participant":   resolvedSender,
			"chatPn":        resolveChatPN(client, v.Info),
			"isGroup":       isGroup,
			"isCommunity":   isCommunity,
			"isSubGroup":    isSubGroup,
			"mentionedJids": mentionedJIDs,
			"quotedMsgId":   quotedMsgID,
			"mimetype":      content.mimeType,
			"forwarded":     forwarded,
			"forwardScore":  fwdScore,
			"rich":          content.rich,
			"fileLength":    content.fileLength,
			// Media is NOT downloaded on receipt (keeps the event loop real-time).
			// Ship the embedded JPEG thumbnail + serialized proto for on-demand DL.
			"thumbnail":  content.thumbnail,
			"mediaProto": content.protoB64,
		},
	})
}

// groupSubject returns the group's display name, cached per group JID to avoid
// a GetGroupInfo round-trip on every message.
func (s *WhatsAppService) groupSubject(client *whatsmeow.Client, jid types.JID) string {
	key := jid.String()
	s.groupNameMu.Lock()
	if n, ok := s.groupNames[key]; ok {
		s.groupNameMu.Unlock()
		return n
	}
	s.groupNameMu.Unlock()

	name := ""
	if info, err := client.GetGroupInfo(context.Background(), jid); err == nil && info != nil {
		name = info.GroupName.Name
		s.groupMetaMu.Lock()
		s.groupMetaMap[key] = groupMeta{
			isCommunity: info.IsParent,
			isSubGroup:  info.IsDefaultSubGroup,
		}
		s.groupMetaMu.Unlock()
	}
	if name != "" {
		s.groupNameMu.Lock()
		s.groupNames[key] = name
		s.groupNameMu.Unlock()
	}
	return name
}

// cachedGroupMeta returns the cached community/sub-group metadata for a group JID.
// Returns zero value (all false) when no info has been cached yet.
func (s *WhatsAppService) cachedGroupMeta(jid types.JID) groupMeta {
	key := jid.String()
	s.groupMetaMu.Lock()
	defer s.groupMetaMu.Unlock()
	return s.groupMetaMap[key]
}

func (s *WhatsAppService) handlePollVoteEvent(client *whatsmeow.Client, id int, tenantID string, v *events.Message) {
	vote, err := client.DecryptPollVote(context.Background(), v)
	if err != nil {
		// Keep the event loop alive; an undecryptable vote is not actionable by business.
		return
	}
	key := v.Message.GetPollUpdateMessage().GetPollCreationMessageKey()
	if key == nil || key.GetID() == "" {
		return
	}
	selected := make([]string, 0, len(vote.GetSelectedOptions()))
	for _, option := range vote.GetSelectedOptions() {
		selected = append(selected, hex.EncodeToString(option))
	}
	s.publishEvent(tenantID, id, "message.poll_vote", map[string]interface{}{
		"sessionId":      fmt.Sprintf("%d", id),
		"pollMessageId":  key.GetID(),
		"voterJid":       v.Info.Sender.String(),
		"optionSelected": strings.Join(selected, ","),
	})
}

func pollVotePayload(id int, pollMessageID, voterJID string, selectedOptions [][]byte) map[string]interface{} {
	selected := make([]string, 0, len(selectedOptions))
	for _, option := range selectedOptions {
		selected = append(selected, hex.EncodeToString(option))
	}
	return map[string]interface{}{
		"sessionId":      idString(id),
		"pollMessageId":  pollMessageID,
		"voterJid":       voterJID,
		"optionSelected": strings.Join(selected, ","),
	}
}

func idString(id int) string { return fmt.Sprintf("%d", id) }

// logRawMessage despeja o protobuf de uma mensagem que ainda não sabemos desenhar, para projetar o suporte a ela com
// o conteúdo real. Desligado por padrão: o corpo pode ter dado pessoal (WATINK_LOG_UNSUPPORTED_RAW=1 só em diagnóstico).
func logRawMessage(id int, msgID string, m *waProto.Message) {
	if os.Getenv("WATINK_LOG_UNSUPPORTED_RAW") != "1" {
		return
	}
	raw := protojson.Format(m)
	if len(raw) > 6000 {
		raw = raw[:6000] + "…"
	}
	log.Printf("[Session-%d] RAW %s: %s", id, msgID, strings.Join(strings.Fields(raw), " "))
}

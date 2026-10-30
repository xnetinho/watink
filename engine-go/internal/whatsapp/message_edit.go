package whatsapp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// editedPayload monta o evento message.edited: o id da mensagem ORIGINAL, o texto novo e o instante da edição.
//
// O texto novo vem do conteúdo editado do ProtocolMessage: texto comum (Conversation/ExtendedText) ou a legenda de uma
// mídia. O instante é o TimestampMS que o aparelho que editou gravou (cai para o da mensagem se faltar): é ele que
// ordena a cadeia no business quando as edições chegam fora de ordem.
func editedPayload(sessionID int, v *events.Message, target string, pm *waProto.ProtocolMessage) map[string]interface{} {
	editedAt := time.UnixMilli(pm.GetTimestampMS())
	if pm.GetTimestampMS() == 0 {
		editedAt = v.Info.Timestamp
	}
	return map[string]interface{}{
		"sessionId": fmt.Sprintf("%d", sessionID),
		"messageId": target,
		"body":      editedBody(pm.GetEditedMessage()),
		"editedAt":  editedAt.UTC().Format(time.RFC3339Nano),
		"fromJid":   v.Info.Sender.String(),
		"fromMe":    v.Info.IsFromMe,
	}
}

// editedBody é o texto do conteúdo editado: o corpo de um texto ou a legenda de uma imagem, vídeo ou documento.
func editedBody(m *waProto.Message) string {
	m = unwrap(m)
	return extractMessageContent(m).body
}

// handleSecretEdit decifra a edição que o contato fez e publica message.edited.
//
// O alvo da edição é targetMessageKey.ID (a mensagem original); Info.ID é o do evento de edição. A chave para decifrar é
// o segredo da original, guardado quando ela chegou: sem ele (mensagem anterior à conexão) não há como ler o texto novo.
func (s *WhatsAppService) handleSecretEdit(client *whatsmeow.Client, id int, tenantID string, v *events.Message) {
	target := v.Message.GetSecretEncryptedMessage().GetTargetMessageKey().GetID()
	decrypted, err := client.DecryptSecretEncryptedMessage(context.Background(), v)
	if err != nil {
		log.Printf("[Session-%d] não consegui decifrar a edição de %s: %v", id, target, err)
		return
	}
	// O conteúdo decifrado é um ProtocolMessage MESSAGE_EDIT: o texto novo está em editedMessage (confirmado com edições
	// reais: {"protocolMessage":{"type":"MESSAGE_EDIT","editedMessage":{"conversation":"..."},"timestampMS":...}}).
	pm := decrypted.GetProtocolMessage()
	if pm == nil || pm.GetType() != waProto.ProtocolMessage_MESSAGE_EDIT {
		log.Printf("[Session-%d] edição de %s com conteúdo inesperado; ignorada", id, target)
		return
	}
	if target == "" {
		target = pm.GetKey().GetID()
	}
	if target == "" {
		return
	}
	s.publishEvent(tenantID, id, "message.edited", editedPayload(id, v, target, pm))
}

// SendEdit edita uma mensagem NOSSA já enviada. A janela do WhatsApp é whatsmeow.EditWindow (20 min): passado o prazo o
// servidor recusa, e o business já barra antes de chegar aqui.
func (s *WhatsAppService) SendEdit(sessionID int, tenantID string, payload EditCommandPayload) error {
	client, err := s.getConnectedClient(sessionID, tenantID)
	if err != nil {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}
	chat, err := ensureJID(payload.To)
	if err != nil {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return fmt.Errorf("invalid JID %q: %w", payload.To, err)
	}
	if payload.TargetMsgID == "" || payload.Body == "" {
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return fmt.Errorf("edit needs targetMsgId and body")
	}

	msg := client.BuildEdit(chat, types.MessageID(payload.TargetMsgID), editedContent(payload))
	// O id do comando NÃO vira o id da mensagem: a edição é um evento sobre a original, que mantém o id dela.
	if _, err := client.SendMessage(context.Background(), chat, msg); err != nil {
		s.reportIfRiskSignal(sessionID, tenantID, "message.edit", err)
		s.emitAck(sessionID, tenantID, payload.MessageID, 5)
		return err
	}
	s.emitAck(sessionID, tenantID, payload.MessageID, 1)
	return nil
}

// editedContent é o conteúdo novo da mensagem: texto simples, ou a LEGENDA quando a original é imagem, vídeo ou
// documento (o WhatsApp não edita o arquivo, só a legenda).
func editedContent(p EditCommandPayload) *waProto.Message {
	switch strings.ToLower(p.MediaType) {
	case "image":
		return &waProto.Message{ImageMessage: &waProto.ImageMessage{Caption: proto.String(p.Body)}}
	case "video":
		return &waProto.Message{VideoMessage: &waProto.VideoMessage{Caption: proto.String(p.Body)}}
	case "document":
		return &waProto.Message{DocumentMessage: &waProto.DocumentMessage{Caption: proto.String(p.Body)}}
	}
	return &waProto.Message{Conversation: proto.String(p.Body)}
}

package whatsapp

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
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

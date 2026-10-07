package whatsapp

import (
	"fmt"
	"log"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// handleUndecryptable trata mensagens que o whatsmeow não conseguiu entregar com conteúdo.
//
// Só a VISUALIZAÇÃO ÚNICA vira mensagem. O WhatsApp não entrega esse conteúdo a aparelhos vinculados
// (só ao celular principal): o que chega é <unavailable type="view_once"/>, sem nenhum <enc>, e o pedido ao
// celular que o whatsmeow faz não traz o conteúdo de volta. Antes o engine ignorava o evento e o atendente
// nem sabia que o cliente mandou algo. Agora o chat recebe um aviso, marcado como "view_once" e sem mídia.
//
// As demais indisponíveis e as falhas comuns de decifragem NÃO viram aviso: o whatsmeow pede o reenvio e a
// mensagem chega depois, normalmente; avisar duplicaria o chat.
func (s *WhatsAppService) handleUndecryptable(_ *whatsmeow.Client, id int, tenantID string, v *events.UndecryptableMessage) {
	if !v.IsUnavailable || v.UnavailableType != events.UnavailableTypeViewOnce {
		return
	}
	// Status/Stories não são conversas (mesma regra do handler de mensagens).
	if v.Info.Chat.String() == "status@broadcast" {
		return
	}

	chat := v.Info.Chat
	sender := v.Info.Sender.ToNonAD()
	isGroup := v.Info.IsGroup || chat.Server == types.GroupServer
	chatJID := chat.String()
	if chatJID == "" {
		chatJID = sender.String()
	}
	pushName := v.Info.PushName
	if v.Info.IsFromMe && !isGroup {
		pushName = ""
	}

	log.Printf("Session %d: visualização única %s de %s registrada sem conteúdo (o WhatsApp não a entrega a aparelhos vinculados)", id, v.Info.ID, chatJID)
	s.publishEvent(tenantID, id, "message.received", map[string]interface{}{
		"sessionId": fmt.Sprintf("%d", id),
		"message": map[string]interface{}{
			"id":            string(v.Info.ID),
			"from":          chatJID,
			"body":          "",
			"type":          "view_once",
			"fromMe":        v.Info.IsFromMe,
			"timestamp":     v.Info.Timestamp.Unix(),
			"pushName":      pushName,
			"groupName":     "",
			"profilePicUrl": "",
			"senderPicUrl":  "",
			"isLid":         v.Info.Sender.Server == types.HiddenUserServer,
			"participant":   sender.String(),
			"chatPn":        "",
			"isGroup":       isGroup,
			"isCommunity":   false,
			"isSubGroup":    false,
			"mentionedJids": []string{},
			"quotedMsgId":   "",
			"mimetype":      "",
			"thumbnail":     "",
			"mediaProto":    "",
		},
	})
}

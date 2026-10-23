package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// resolveChatPN devolve o JID de telefone ("5511...@s.whatsapp.net") do chat de
// uma conversa 1:1, ou "" quando o chat não é individual ou o telefone não é
// conhecido.
//
// O WhatsApp entrega conversas individuais com o chat como LID ("...@lid") e
// guarda o mapa LID<->telefone no Store. O business precisa do telefone para
// reconhecer o contato que o atendente já tinha na agenda (cadastrado pelo
// número) em vez de criar um segundo contato com o LID como se fosse número.
//
// Ordem: mapa do Store; depois o par alternativo que o próprio evento traz
// (RecipientAlt quando fui eu quem enviou, SenderAlt quando a pessoa enviou).
func resolveChatPN(client *whatsmeow.Client, info types.MessageInfo) string {
	if info.IsGroup || info.Chat.Server != types.HiddenUserServer {
		return ""
	}

	if client != nil && client.Store != nil && client.Store.LIDs != nil {
		if pn, err := client.Store.LIDs.GetPNForLID(context.Background(), info.Chat); err == nil && !pn.IsEmpty() {
			return pn.ToNonAD().String()
		}
	}

	alt := info.SenderAlt
	if info.IsFromMe {
		alt = info.RecipientAlt
	}
	if !alt.IsEmpty() && alt.Server == types.DefaultUserServer {
		return alt.ToNonAD().String()
	}
	return ""
}

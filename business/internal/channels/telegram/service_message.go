package telegram

import "encoding/json"

// telegramServiceFields são as mensagens de SERVIÇO do Telegram: avisos do próprio chat (entrada e saída de membro, título, foto, fixação, tópicos, chamada de
// vídeo...), sem conteúdo de atendimento. Não viram mensagem nem ticket: no grupo o aviso "entrou fulano" criava ou sujava um ticket. Lista fechada: um campo
// novo que o Telegram criar continua aparecendo como "não suportado" em vez de sumir.
var telegramServiceFields = map[string]bool{
	"new_chat_members": true, "new_chat_participant": true, "new_chat_member": true, "left_chat_member": true, "left_chat_participant": true,
	"new_chat_title": true, "new_chat_photo": true, "delete_chat_photo": true, "group_chat_created": true, "supergroup_chat_created": true,
	"channel_chat_created": true, "message_auto_delete_timer_changed": true, "migrate_to_chat_id": true, "migrate_from_chat_id": true,
	"pinned_message": true, "forum_topic_created": true, "forum_topic_edited": true, "forum_topic_closed": true, "forum_topic_reopened": true,
	"general_forum_topic_hidden": true, "general_forum_topic_unhidden": true, "video_chat_scheduled": true, "video_chat_started": true,
	"video_chat_ended": true, "video_chat_participants_invited": true, "boost_added": true, "chat_background_set": true,
	"write_access_allowed": true, "proximity_alert_triggered": true,
}

func isTelegramServiceMessage(m *TelegramMessage, raw []byte) bool {
	if m == nil {
		return false
	}
	var u map[string]json.RawMessage
	if json.Unmarshal(raw, &u) != nil {
		return false
	}
	for _, key := range []string{"message", "edited_message"} {
		var fields map[string]json.RawMessage
		if json.Unmarshal(u[key], &fields) != nil {
			continue
		}
		for k, v := range fields {
			if telegramServiceFields[k] && string(v) != "null" {
				return true
			}
		}
	}
	return false
}

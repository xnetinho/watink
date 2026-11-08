package telegram

import (
	"fmt"
	"strconv"

	"github.com/alltomatos/watinkdev/business/internal/domain"
)

// telegramChatIDOf devolve o chat_id (texto) que a Bot API espera. A chave da conversa tem de ser do Telegram ("tg:<chat_id>"); sem chave,
// cai no ExternalID, que o painel manda já como chat_id. Qualquer outra coisa é recusada em vez de ir à Bot API como se fosse um chat.
func telegramChatIDOf(conversationKey, externalID string) (string, error) {
	if conversationKey != "" {
		id, ok := domain.TelegramChatID(conversationKey)
		if !ok {
			return "", fmt.Errorf("telegram: chave de conversa %q não é do Telegram (esperado tg:<chat_id>)", conversationKey)
		}
		return strconv.FormatInt(id, 10), nil
	}
	if externalID == "" {
		return "", fmt.Errorf("telegram: conversa sem chat_id")
	}
	return externalID, nil
}

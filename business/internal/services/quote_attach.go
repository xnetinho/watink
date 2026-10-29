package services

import (
	"context"

	"github.com/alltomatos/watinkdev/business/internal/domain"
)

// attachQuote anexa a mensagem citada antes de anunciar a mensagem em tempo real. Sem isto a resposta recebida chegava
// à tela como mensagem comum e só virava resposta ao recarregar o chat (a listagem é quem anexava a citação).
func attachQuote(ctx context.Context, repo domain.MessageRepository, msg *domain.Message) {
	if msg == nil || msg.QuotedMsgID == nil || *msg.QuotedMsgID == "" {
		return
	}
	if q, err := repo.FindByID(ctx, *msg.QuotedMsgID, msg.TenantID); err == nil {
		msg.QuotedMsg = q
	}
}

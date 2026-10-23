package domain

import (
	"fmt"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
)

// ReplyPolicy é a avaliação da permissão de envio para um ticket/conversa (D2 / ADR 0033).
type ReplyPolicy struct {
	CanReply     bool             `json:"canReply"`
	Reason       ReplyBlockReason `json:"reason"`
	Message      string           `json:"message,omitempty"`
	ValidUntil   *time.Time       `json:"validUntil,omitempty"`
	Capabilities Capabilities     `json:"capabilities"`
}

// EvaluateReplyPolicy avalia se o atendente/sistema pode responder no ticket dado o canal e o histórico.
func EvaluateReplyPolicy(
	caps Capabilities,
	conn *models.ChannelConnection,
	lastInboundAt *time.Time,
	now time.Time,
) ReplyPolicy {
	policy := ReplyPolicy{
		CanReply:     true,
		Reason:       ReplyBlockNone,
		Capabilities: caps,
	}

	// 1. Conexão desconectada
	if conn != nil && conn.Status != "CONNECTED" && conn.Status != "connected" {
		policy.CanReply = false
		policy.Reason = ReplyBlockDisconnected
		policy.Message = "A conexão deste canal está desconectada."
		return policy
	}

	// 2. Canal que não permite iniciar conversa sem mensagem prévia
	if !caps.InitiateConversation && lastInboundAt == nil {
		policy.CanReply = false
		policy.Reason = ReplyBlockInitiatorNotAllowed
		policy.Message = "Este canal não permite iniciar conversas com o cliente."
		return policy
	}

	// 3. Janela de resposta expirada (ex.: Instagram 24h, Mercado Livre 48h)
	if caps.ReplyWindow != nil && lastInboundAt != nil {
		windowEnd := lastInboundAt.Add(*caps.ReplyWindow)
		if now.After(windowEnd) {
			policy.CanReply = false
			policy.Reason = ReplyBlockWindowExpired
			policy.ValidUntil = &windowEnd
			hours := int(caps.ReplyWindow.Hours())
			policy.Message = fmt.Sprintf("A janela de resposta de %d horas deste canal expirou.", hours)
			return policy
		}
		policy.ValidUntil = &windowEnd
	}

	return policy
}

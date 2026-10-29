package services

import (
	"context"
	"fmt"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
)

type MessageDispatchService struct {
	registry *domain.ChannelRegistry
}

func NewMessageDispatchService(registry *domain.ChannelRegistry) *MessageDispatchService {
	if registry == nil {
		registry = domain.DefaultChannelRegistry()
	}
	return &MessageDispatchService{
		registry: registry,
	}
}

// ValidateOutbound valida limites de texto e suporte a mídia contra os Capabilities do canal
func (s *MessageDispatchService) ValidateOutbound(caps domain.Capabilities, msg domain.OutboundMessage) error {
	// 1. Limite de texto
	if caps.MaxTextBytes > 0 && len([]byte(msg.Body)) > caps.MaxTextBytes {
		return fmt.Errorf("mensagem excede o limite maximo de %d bytes do canal (enviado: %d bytes)", caps.MaxTextBytes, len([]byte(msg.Body)))
	}

	// 2. Validação de mídia
	if msg.MediaType != "" {
		supported := false
		for _, m := range caps.Media {
			if string(m) == msg.MediaType {
				supported = true
				break
			}
		}
		if !supported {
			return fmt.Errorf("tipo de midia %q nao suportado pelo canal", msg.MediaType)
		}
	}

	return nil
}

// Dispatch envia uma mensagem respeitando ReplyPolicy e Capabilities do canal
func (s *MessageDispatchService) Dispatch(
	ctx context.Context,
	conn *models.ChannelConnection,
	msg domain.OutboundMessage,
	lastInboundAt *time.Time,
	now time.Time,
) (domain.SendResult, error) {
	if conn == nil {
		return domain.SendResult{}, fmt.Errorf("channel connection is nil")
	}

	adapter, err := s.registry.Get(domain.ChannelKind(conn.Kind))
	if err != nil {
		return domain.SendResult{}, err
	}

	caps := adapter.Capabilities()

	// 1. Validar ReplyPolicy
	policy := domain.EvaluateReplyPolicy(caps, conn, lastInboundAt, now)
	if !policy.CanReply {
		return domain.SendResult{}, fmt.Errorf("envio bloqueado por politica do canal: %s (%s)", policy.Reason, policy.Message)
	}

	// 2. Validar Capabilities (tamanho de texto e tipo de mídia)
	if err := s.ValidateOutbound(caps, msg); err != nil {
		return domain.SendResult{}, err
	}

	// 3. Enviar via adaptador do canal
	return adapter.Send(ctx, conn, msg)
}

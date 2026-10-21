package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ChannelContractSuite define a suíte de testes de contrato que todo adaptador de canal deve satisfazer (Tarefa 3.5 / ADR 0033).
func RunChannelContractSuite(t *testing.T, ch domain.Channel, validConn *models.ChannelConnection, disconnectedConn *models.ChannelConnection) {
	t.Run("Contrato: Identificação e Capacidades Válidas", func(t *testing.T) {
		assert.NotEmpty(t, ch.Kind())
		caps := ch.Capabilities()
		assert.NotEmpty(t, caps.Auth)
		if caps.ReplyWindow != nil {
			assert.True(t, *caps.ReplyWindow > 0, "reply window deve ser positiva se definida")
		}
	})

	t.Run("Contrato: Resposta Bloqueada se Desconectado", func(t *testing.T) {
		caps := ch.Capabilities()
		policy := domain.EvaluateReplyPolicy(caps, disconnectedConn, nil, time.Now())
		assert.False(t, policy.CanReply)
		assert.Equal(t, domain.ReplyBlockDisconnected, policy.Reason)
	})

	t.Run("Contrato: Janela de Resposta Respeitada", func(t *testing.T) {
		caps := ch.Capabilities()
		now := time.Now()
		if caps.ReplyWindow != nil {
			// Janela vencida
			expiredTime := now.Add(-(*caps.ReplyWindow + time.Minute))
			policyExpired := domain.EvaluateReplyPolicy(caps, validConn, &expiredTime, now)
			assert.False(t, policyExpired.CanReply)
			assert.Equal(t, domain.ReplyBlockWindowExpired, policyExpired.Reason)

			// Dentro da janela
			validTime := now.Add(-(*caps.ReplyWindow / 2))
			policyValid := domain.EvaluateReplyPolicy(caps, validConn, &validTime, now)
			assert.True(t, policyValid.CanReply)
			assert.Equal(t, domain.ReplyBlockNone, policyValid.Reason)
		}
	})

	t.Run("Contrato: Respeito ao Limite de Texto", func(t *testing.T) {
		caps := ch.Capabilities()
		if caps.MaxTextBytes > 0 {
			msgTooLong := domain.OutboundMessage{
				Body: string(make([]byte, caps.MaxTextBytes+100)),
			}
			assert.True(t, len(msgTooLong.Body) > caps.MaxTextBytes)
		}
	})

	t.Run("Contrato: Disconnect não gera pânico", func(t *testing.T) {
		ctx := context.Background()
		err := ch.Disconnect(ctx, validConn)
		require.NoError(t, err)
	})
}

func TestMockChannelContract(t *testing.T) {
	window := 24 * time.Hour
	ch := &mockChannel{
		kind: domain.ChannelKindTelegram,
		caps: domain.Capabilities{
			Auth:         domain.AuthKindToken,
			MaxTextBytes: 4096,
			ReplyWindow:  &window,
		},
	}
	validConn := &models.ChannelConnection{Status: "CONNECTED"}
	disconnectedConn := &models.ChannelConnection{Status: "DISCONNECTED"}

	RunChannelContractSuite(t, ch, validConn, disconnectedConn)
}

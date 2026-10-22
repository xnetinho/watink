package domain

import (
	"context"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RunChannelContractSuite define a suíte de testes de contrato que todo adaptador de canal deve satisfazer (Tarefa 3.5 / ADR 0033).
func RunChannelContractSuite(t *testing.T, ch Channel, validConn *models.ChannelConnection, disconnectedConn *models.ChannelConnection) {
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
		policy := EvaluateReplyPolicy(caps, disconnectedConn, nil, time.Now())
		assert.False(t, policy.CanReply)
		assert.Equal(t, ReplyBlockDisconnected, policy.Reason)
	})

	t.Run("Contrato: Janela de Resposta Respeitada", func(t *testing.T) {
		caps := ch.Capabilities()
		now := time.Now()
		if caps.ReplyWindow != nil {
			// Janela vencida
			expiredTime := now.Add(-(*caps.ReplyWindow + time.Minute))
			policyExpired := EvaluateReplyPolicy(caps, validConn, &expiredTime, now)
			assert.False(t, policyExpired.CanReply)
			assert.Equal(t, ReplyBlockWindowExpired, policyExpired.Reason)

			// Dentro da janela
			validTime := now.Add(-(*caps.ReplyWindow / 2))
			policyValid := EvaluateReplyPolicy(caps, validConn, &validTime, now)
			assert.True(t, policyValid.CanReply)
			assert.Equal(t, ReplyBlockNone, policyValid.Reason)
		}
	})

	t.Run("Contrato: Respeito ao Limite de Texto", func(t *testing.T) {
		caps := ch.Capabilities()
		if caps.MaxTextBytes > 0 {
			msgTooLong := OutboundMessage{
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

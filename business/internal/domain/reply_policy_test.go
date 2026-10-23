package domain_test

import (
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateReplyPolicy(t *testing.T) {
	now := time.Now()
	window24h := 24 * time.Hour

	t.Run("WhatsApp sem janela e conectado permite resposta", func(t *testing.T) {
		caps := domain.Capabilities{
			InitiateConversation: true,
			ReplyWindow:          nil,
		}
		conn := &models.ChannelConnection{Status: "CONNECTED"}
		policy := domain.EvaluateReplyPolicy(caps, conn, nil, now)
		assert.True(t, policy.CanReply)
		assert.Equal(t, domain.ReplyBlockNone, policy.Reason)
	})

	t.Run("Canal desconectado bloqueia resposta", func(t *testing.T) {
		caps := domain.Capabilities{InitiateConversation: true}
		conn := &models.ChannelConnection{Status: "DISCONNECTED"}
		policy := domain.EvaluateReplyPolicy(caps, conn, nil, now)
		assert.False(t, policy.CanReply)
		assert.Equal(t, domain.ReplyBlockDisconnected, policy.Reason)
	})

	t.Run("Canal que não inicia conversa sem inbound bloqueia", func(t *testing.T) {
		caps := domain.Capabilities{InitiateConversation: false}
		conn := &models.ChannelConnection{Status: "CONNECTED"}
		policy := domain.EvaluateReplyPolicy(caps, conn, nil, now)
		assert.False(t, policy.CanReply)
		assert.Equal(t, domain.ReplyBlockInitiatorNotAllowed, policy.Reason)
	})

	t.Run("Janela de 24h expirada bloqueia", func(t *testing.T) {
		caps := domain.Capabilities{
			InitiateConversation: false,
			ReplyWindow:          &window24h,
		}
		conn := &models.ChannelConnection{Status: "CONNECTED"}
		lastInbound := now.Add(-25 * time.Hour)
		policy := domain.EvaluateReplyPolicy(caps, conn, &lastInbound, now)
		assert.False(t, policy.CanReply)
		assert.Equal(t, domain.ReplyBlockWindowExpired, policy.Reason)
	})

	t.Run("Janela de 24h dentro do prazo permite", func(t *testing.T) {
		caps := domain.Capabilities{
			InitiateConversation: false,
			ReplyWindow:          &window24h,
		}
		conn := &models.ChannelConnection{Status: "CONNECTED"}
		lastInbound := now.Add(-10 * time.Hour)
		policy := domain.EvaluateReplyPolicy(caps, conn, &lastInbound, now)
		assert.True(t, policy.CanReply)
		assert.Equal(t, domain.ReplyBlockNone, policy.Reason)
	})
}

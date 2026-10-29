package services

import (
	"context"
	"errors"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAttachQuote(t *testing.T) {
	id := "ORIG"
	repo := &ackMockMessageRepo{msg: &domain.Message{ID: "ORIG", MediaType: "audio"}}

	t.Run("resposta recebida chega à tela já com a citada", func(t *testing.T) {
		m := &domain.Message{ID: "R1", QuotedMsgID: &id, TenantID: uuid.New()}
		attachQuote(context.Background(), repo, m)
		if assert.NotNil(t, m.QuotedMsg) {
			assert.Equal(t, "audio", m.QuotedMsg.MediaType)
		}
	})
	t.Run("sem citação, ou citada que sumiu, não quebra nem inventa", func(t *testing.T) {
		plain := &domain.Message{ID: "R2"}
		attachQuote(context.Background(), repo, plain)
		assert.Nil(t, plain.QuotedMsg)
		attachQuote(context.Background(), &ackMockMessageRepo{findErr: errors.New("não achou")}, &domain.Message{QuotedMsgID: &id})
		attachQuote(context.Background(), repo, nil)
	})
}

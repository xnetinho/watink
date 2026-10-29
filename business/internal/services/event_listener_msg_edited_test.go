package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func editedEvent(t *testing.T, id, body string, at time.Time, fromMe bool) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(map[string]interface{}{
		"sessionId": "1", "messageId": id, "body": body, "editedAt": at.UTC().Format(time.RFC3339Nano), "fromMe": fromMe,
	})
	require.NoError(t, err)
	return b
}

func newEditListener(e *editEnv) (*EventListener, *wdBroadcast) {
	b := &wdBroadcast{}
	return &EventListener{db: e.db, broadcast: b}, b
}

func TestHandleMessageEdited(t *testing.T) {
	e := newEditEnv(t)
	el, b := newEditListener(e)
	ctx := context.Background()

	t.Run("a edição do contato entra na cadeia e a tela é avisada", func(t *testing.T) {
		require.NoError(t, el.handleMessageEdited(ctx, editedEvent(t, "M1", "Olá", e.t0.Add(time.Minute), false), e.tenant))
		assert.Equal(t, []string{"Oi", "Olá"}, bodies(e.chain(t)))
		assert.Equal(t, "Olá", e.message(t).Body)
		assert.Equal(t, models.EditedByContact, e.chain(t)[1].EditedBy)
		require.Len(t, b.events, 1)
		assert.Equal(t, "appMessage", b.events[0])
	})

	t.Run("o mesmo evento reentregue não duplica nem avisa de novo", func(t *testing.T) {
		require.NoError(t, el.handleMessageEdited(ctx, editedEvent(t, "M1", "Olá", e.t0.Add(time.Minute), false), e.tenant))
		assert.Len(t, e.chain(t), 2)
		assert.Len(t, b.events, 1)
	})

	t.Run("uma segunda edição acrescenta a versão seguinte", func(t *testing.T) {
		require.NoError(t, el.handleMessageEdited(ctx, editedEvent(t, "M1", "Olá, tudo bem?", e.t0.Add(2*time.Minute), false), e.tenant))
		assert.Equal(t, []string{"Oi", "Olá", "Olá, tudo bem?"}, bodies(e.chain(t)))
		assert.Equal(t, 2, e.message(t).EditCount)
	})

	t.Run("edição feita por nós pelo celular é registrada como nossa", func(t *testing.T) {
		require.NoError(t, el.handleMessageEdited(ctx, editedEvent(t, "M1", "pelo celular", e.t0.Add(3*time.Minute), true), e.tenant))
		c := e.chain(t)
		assert.Equal(t, models.EditedByMe, c[len(c)-1].EditedBy)
	})

	t.Run("original que não está no Watink: ignora sem erro (não pode travar a fila do broker)", func(t *testing.T) {
		before := len(b.events)
		require.NoError(t, el.handleMessageEdited(ctx, editedEvent(t, "NAO_EXISTE", "x", e.t0, false), e.tenant))
		assert.Len(t, b.events, before)
	})

	t.Run("edição sem texto: ignora e a mensagem continua com o texto", func(t *testing.T) {
		before := e.message(t)
		require.NoError(t, el.handleMessageEdited(ctx, editedEvent(t, "M1", "", e.t0.Add(30*time.Minute), false), e.tenant))
		after := e.message(t)
		assert.Equal(t, before.Body, after.Body)
		assert.Equal(t, before.EditCount, after.EditCount)
	})

	t.Run("evento sem id da mensagem ou JSON inválido", func(t *testing.T) {
		require.NoError(t, el.handleMessageEdited(ctx, editedEvent(t, "", "x", e.t0, false), e.tenant))
		assert.Error(t, el.handleMessageEdited(ctx, json.RawMessage(`{x`), e.tenant))
	})

	t.Run("horário inválido não perde a edição: usa o instante do recebimento", func(t *testing.T) {
		raw, _ := json.Marshal(map[string]interface{}{"messageId": "M1", "body": "sem hora", "editedAt": "lixo"})
		require.NoError(t, el.handleMessageEdited(ctx, raw, e.tenant))
		assert.Equal(t, "sem hora", e.message(t).Body)
	})
}

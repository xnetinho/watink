package telegram

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramAdapter_ResolveAddress(t *testing.T) {
	adapter := NewAdapter(nil)
	conn := &models.ChannelConnection{Kind: "telegram"}
	contact := &models.Contact{Number: "tg_9999"}

	addr, err := adapter.ResolveAddress(context.Background(), conn, contact)
	require.NoError(t, err)
	assert.Equal(t, "tg_9999", addr)
}

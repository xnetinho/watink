package whatsapp

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhatsAppAdapter_ResolveAddress(t *testing.T) {
	adapter := NewAdapter(nil)
	conn := &models.ChannelConnection{Kind: "whatsapp"}

	// 1. Número comum -> @s.whatsapp.net
	c1 := &models.Contact{Number: "5511999999999"}
	addr1, err := adapter.ResolveAddress(context.Background(), conn, c1)
	require.NoError(t, err)
	assert.Equal(t, "5511999999999@s.whatsapp.net", addr1)

	// 2. LID prioritário
	lid := "12345@lid"
	c2 := &models.Contact{Number: "5511999999999", Lid: &lid}
	addr2, err := adapter.ResolveAddress(context.Background(), conn, c2)
	require.NoError(t, err)
	assert.Equal(t, "12345@lid", addr2)
}

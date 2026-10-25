package fake_test

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/channels/fake"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeChannel_EndToEndAgnostic(t *testing.T) {
	reg := domain.NewChannelRegistry()
	fakeChan := fake.NewChannel()
	reg.Register(fakeChan)

	ch, err := reg.Get(fake.ChannelKindFake)
	require.NoError(t, err)
	assert.Equal(t, fake.ChannelKindFake, ch.Kind())

	conn := &models.ChannelConnection{
		Kind:   string(fake.ChannelKindFake),
		Status: "CONNECTED",
	}

	// 1. Connect
	res, err := ch.Connect(context.Background(), conn, domain.ConnectInput{})
	require.NoError(t, err)
	assert.True(t, res.Connected)

	// 2. Resolve Address
	addr, err := ch.ResolveAddress(context.Background(), conn, &models.Contact{Number: "fake_user"})
	require.NoError(t, err)
	assert.Equal(t, "fake_user", addr)

	// 3. Send
	sendRes, err := ch.Send(context.Background(), conn, domain.OutboundMessage{Body: "Ola canal fake!"})
	require.NoError(t, err)
	assert.Equal(t, "sent", sendRes.Status)
	assert.Equal(t, 1, fakeChan.GetSentCount())

	// 4. Inbound Parse
	inbound, err := ch.ParseInbound(context.Background(), conn, []byte("Mensagem recebida fake"), nil)
	require.NoError(t, err)
	require.Len(t, inbound, 1)
	assert.Equal(t, "Mensagem recebida fake", inbound[0].Body)

	// 5. Disconnect
	err = ch.Disconnect(context.Background(), conn)
	require.NoError(t, err)
}

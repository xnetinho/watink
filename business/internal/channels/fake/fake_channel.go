package fake

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
)

const ChannelKindFake domain.ChannelKind = "fake"

type Channel struct {
	mu       sync.Mutex
	sentMsgs []domain.OutboundMessage
}

func NewChannel() *Channel {
	return &Channel{
		sentMsgs: make([]domain.OutboundMessage, 0),
	}
}

func (c *Channel) Kind() domain.ChannelKind {
	return ChannelKindFake
}

func (c *Channel) Capabilities() domain.Capabilities {
	return domain.Capabilities{
		InitiateConversation: true,
		MaxTextBytes:         4096,
		Media:                []domain.MediaKind{domain.MediaKindImage, domain.MediaKindVideo},
		Typing:               true,
		ReadReceipts:         true,
	}
}

func (c *Channel) Connect(ctx context.Context, conn *models.ChannelConnection, input domain.ConnectInput) (domain.ConnectResult, error) {
	return domain.ConnectResult{Status: "CONNECTED", Connected: true}, nil
}

func (c *Channel) Disconnect(ctx context.Context, conn *models.ChannelConnection) error {
	return nil
}

func (c *Channel) Send(ctx context.Context, conn *models.ChannelConnection, msg domain.OutboundMessage) (domain.SendResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sentMsgs = append(c.sentMsgs, msg)
	return domain.SendResult{
		ExternalMessageID: fmt.Sprintf("fake_msg_%d", len(c.sentMsgs)),
		Timestamp:         time.Now(),
		Status:            "sent",
	}, nil
}

func (c *Channel) ResolveAddress(ctx context.Context, conn *models.ChannelConnection, contact *models.Contact) (string, error) {
	if contact == nil {
		return "", fmt.Errorf("contact nil")
	}
	return contact.Number, nil
}

func (c *Channel) ParseInbound(ctx context.Context, conn *models.ChannelConnection, raw []byte, headers map[string]string) ([]domain.InboundMessage, error) {
	return []domain.InboundMessage{
		{
			ExternalMessageID: "fake_in_1",
			Body:              string(raw),
			ConversationKey:   "fake_conv_1",
		},
	}, nil
}

func (c *Channel) GetSentCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.sentMsgs)
}

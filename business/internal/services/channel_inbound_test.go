package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/application/usecases"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type mockRedisService struct {
	locks map[string]bool
}

func newMockRedis() *mockRedisService {
	return &mockRedisService{locks: make(map[string]bool)}
}

func (m *mockRedisService) SetLock(key string, _ string, _ time.Duration) (bool, error) {
	if m.locks[key] {
		return false, nil // Já existe lock
	}
	m.locks[key] = true
	return true, nil
}

func (m *mockRedisService) DelLock(key string) error {
	delete(m.locks, key)
	return nil
}

func (m *mockRedisService) Get(_ context.Context, _ string) (string, error)          { return "", nil }
func (m *mockRedisService) Subscribe(_ context.Context, _ string) *redis.PubSub      { return nil }
func (m *mockRedisService) Publish(_ context.Context, _ string, _ interface{}) error { return nil }
func (m *mockRedisService) Ping(_ context.Context) error                             { return nil }

func TestChannelInboundService_DedupAndAgnosticFlow(t *testing.T) {
	db := testutil.NewTestDB(t)
	redis := newMockRedis()

	tenantID := uuid.New()
	tenant := models.Tenant{ID: tenantID, Name: "Tenant Channel Inbound"}
	require.NoError(t, db.Create(&tenant).Error)

	conn := models.ChannelConnection{
		TenantID: tenantID,
		Kind:     string(domain.ChannelKindTelegram),
		Name:     "Telegram Atendimento",
		Status:   "CONNECTED",
	}
	require.NoError(t, db.Create(&conn).Error)

	ticketRepo := repository.NewGORMTicketRepo(db)
	contactRepo := repository.NewGORMContactRepo(db)
	messageRepo := repository.NewGORMMessageRepo(db)
	receiveUC := usecases.NewReceiveMessageUseCase(nil, messageRepo, contactRepo, ticketRepo, nil, nil, nil)

	inboundService := services.NewChannelInboundService(
		db,
		redis,
		ticketRepo,
		contactRepo,
		messageRepo,
		receiveUC,
		nil,
		nil,
	)

	ctx := context.Background()
	msg := domain.InboundMessage{
		TenantID:            tenantID,
		ChannelConnectionID: conn.ID,
		Channel:             domain.ChannelKindTelegram,
		ExternalMessageID:   "tg-msg-9999",
		ConversationKey:     "tg-chat-8888",
		Sender: domain.InboundSender{
			ExternalID: "tg-chat-8888",
			Name:       "Cliente Telegram",
			AvatarURL:  "https://avatar.url/tg.jpg",
		},
		Body:      "Olá suporte, preciso de ajuda",
		MediaType: "chat",
		Timestamp: time.Now(),
	}

	// 1. Primeira entrega da mensagem
	res, err := inboundService.Receive(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "Cliente Telegram", res.Contact.Name)
	assert.Equal(t, "tg-msg-9999", res.Message.ExternalID)
	assert.Equal(t, domain.StorageMessageID(domain.ChannelKindTelegram, conn.ID, "tg-chat-8888", "tg-msg-9999"), res.Message.ID)
	assert.Equal(t, "Olá suporte, preciso de ajuda", res.Message.Body)
	assert.Equal(t, "pending", res.Ticket.Status)

	// Verifica se a ContactIdentity foi criada
	var identity models.ContactIdentity
	err = db.Where(`"tenantId" = ? AND "channel" = ? AND "externalId" = ?`, tenantID, "telegram", "tg-chat-8888").First(&identity).Error
	require.NoError(t, err)
	assert.Equal(t, res.Contact.ID, identity.ContactID)

	// 2. Segunda entrega com o mesmo ID (dedup no Redis) deve ser ignorada sem erro
	resDup, err := inboundService.Receive(ctx, msg)
	require.NoError(t, err)
	assert.Nil(t, resDup, "mensagem duplicada deve ser descartada pelo dedup do Redis")
}

// failingTicketRepo falha na busca de ticket para exercitar o caminho de erro.
type failingTicketRepo struct {
	domain.TicketRepository
	err error
}

func (f failingTicketRepo) FindOpenByConversation(context.Context, uuid.UUID, int, string) (*domain.Ticket, error) {
	return nil, f.err
}

func (f failingTicketRepo) FindOrCreatePending(context.Context, *domain.Ticket) (*domain.Ticket, error) {
	return nil, f.err
}

func TestChannelInboundService_ReleasesDedupOnFailure(t *testing.T) {
	db := testutil.NewTestDB(t)
	redis := newMockRedis()

	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "Tenant Falha"}).Error)

	conn := models.ChannelConnection{
		TenantID: tenantID,
		Kind:     string(domain.ChannelKindTelegram),
		Name:     "Telegram Falha",
		Status:   "CONNECTED",
	}
	require.NoError(t, db.Create(&conn).Error)

	inboundService := services.NewChannelInboundService(
		db,
		redis,
		failingTicketRepo{err: errTicketRepoDown},
		repository.NewGORMContactRepo(db),
		repository.NewGORMMessageRepo(db),
		nil,
		nil,
		nil,
	)

	msg := domain.InboundMessage{
		TenantID:            tenantID,
		ChannelConnectionID: conn.ID,
		Channel:             domain.ChannelKindTelegram,
		ExternalMessageID:   "tg-msg-fail-1",
		ConversationKey:     "tg-chat-fail-1",
		Sender:              domain.InboundSender{ExternalID: "tg-chat-fail-1", Name: "Cliente"},
		Body:                "mensagem que falha",
		MediaType:           "chat",
		Timestamp:           time.Now(),
	}

	_, err := inboundService.Receive(context.Background(), msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), errTicketRepoDown.Error())

	assert.Empty(t, redis.locks, "lock de dedup deve ser liberado quando o processamento falha")

	// A reentrega não pode ser descartada como duplicata: ela falha de novo (não retorna nil).
	_, err = inboundService.Receive(context.Background(), msg)
	require.Error(t, err)
}

var errTicketRepoDown = errors.New("ticket repo down")

func newInboundSvc(t *testing.T) (*services.ChannelInboundService, *gorm.DB, uuid.UUID, models.ChannelConnection) {
	t.Helper()
	db := testutil.NewTestDB(t)
	tenantID := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenantID, Name: "Tenant Colisao"}).Error)
	conn := models.ChannelConnection{TenantID: tenantID, Kind: string(domain.ChannelKindTelegram), Name: "TG", Status: "CONNECTED"}
	require.NoError(t, db.Create(&conn).Error)
	svc := services.NewChannelInboundService(db, newMockRedis(), repository.NewGORMTicketRepo(db), repository.NewGORMContactRepo(db),
		repository.NewGORMMessageRepo(db), nil, nil, nil)
	return svc, db, tenantID, conn
}

func tgMsg(tenantID uuid.UUID, connID int, chat, extID, body string) domain.InboundMessage {
	return domain.InboundMessage{
		TenantID: tenantID, ChannelConnectionID: connID, Channel: domain.ChannelKindTelegram,
		ExternalMessageID: extID, ConversationKey: chat,
		Sender: domain.InboundSender{ExternalID: chat, Name: "Cliente " + chat},
		Body:   body, MediaType: "chat", Timestamp: time.Now(),
	}
}

// No Telegram o message_id é um contador por chat: o mesmo "42" existe em todo chat. Antes, a segunda mensagem era
// descartada em silêncio pelo ON CONFLICT DO NOTHING sobre a chave primária global.
func TestChannelInbound_MesmoMessageIDEmDoisChatsGravaAsDuas(t *testing.T) {
	svc, db, tenantID, conn := newInboundSvc(t)
	ctx := context.Background()

	a, err := svc.Receive(ctx, tgMsg(tenantID, conn.ID, "chat-A", "42", "mensagem do chat A"))
	require.NoError(t, err)
	b, err := svc.Receive(ctx, tgMsg(tenantID, conn.ID, "chat-B", "42", "mensagem do chat B"))
	require.NoError(t, err)
	require.NotNil(t, a)
	require.NotNil(t, b)

	assert.NotEqual(t, a.Message.ID, b.Message.ID)
	assert.NotEqual(t, a.Ticket.ID, b.Ticket.ID)
	var n int64
	require.NoError(t, db.Model(&models.Message{}).Where(`"tenantId" = ?`, tenantID).Count(&n).Error)
	assert.EqualValues(t, 2, n, "as duas mensagens precisam ser gravadas")

	var stored models.Message
	require.NoError(t, db.Where("id = ?", b.Message.ID).First(&stored).Error)
	assert.Equal(t, "mensagem do chat B", stored.Body)
	assert.Equal(t, "42", stored.ExternalID)
	assert.Equal(t, b.Ticket.ID, stored.TicketID)
}

func TestChannelInbound_MesmoMessageIDEmConexoesDiferentesNaoColide(t *testing.T) {
	svc, db, tenantID, conn1 := newInboundSvc(t)
	conn2 := models.ChannelConnection{TenantID: tenantID, Kind: string(domain.ChannelKindTelegram), Name: "TG2", Status: "CONNECTED"}
	require.NoError(t, db.Create(&conn2).Error)
	ctx := context.Background()

	_, err := svc.Receive(ctx, tgMsg(tenantID, conn1.ID, "chat-A", "7", "bot 1"))
	require.NoError(t, err)
	_, err = svc.Receive(ctx, tgMsg(tenantID, conn2.ID, "chat-A", "7", "bot 2"))
	require.NoError(t, err)

	var n int64
	require.NoError(t, db.Model(&models.Message{}).Where(`"tenantId" = ?`, tenantID).Count(&n).Error)
	assert.EqualValues(t, 2, n)
}

// A reentrega do MESMO evento continua idempotente: uma linha só.
func TestChannelInbound_ReentregaNaoDuplica(t *testing.T) {
	svc, db, tenantID, conn := newInboundSvc(t)
	ctx := context.Background()
	m := tgMsg(tenantID, conn.ID, "chat-A", "42", "oi")

	_, err := svc.Receive(ctx, m)
	require.NoError(t, err)
	// Redis novo (sem a marca de dedup) força o caminho do banco, como numa reentrega após a marca expirar.
	svc2 := services.NewChannelInboundService(db, newMockRedis(), repository.NewGORMTicketRepo(db), repository.NewGORMContactRepo(db),
		repository.NewGORMMessageRepo(db), nil, nil, nil)
	_, err = svc2.Receive(ctx, m)
	require.NoError(t, err)

	var n int64
	require.NoError(t, db.Model(&models.Message{}).Where(`"tenantId" = ?`, tenantID).Count(&n).Error)
	assert.EqualValues(t, 1, n)
}

func TestStorageMessageID_WhatsAppMantemOIdCru(t *testing.T) {
	assert.Equal(t, "3EB0ABC", domain.StorageMessageID(domain.ChannelKindWhatsApp, 5, "5511@s.whatsapp.net", "3EB0ABC"))
	assert.Equal(t, "telegram:5:chat-A:42", domain.StorageMessageID(domain.ChannelKindTelegram, 5, "chat-A", "42"))
}

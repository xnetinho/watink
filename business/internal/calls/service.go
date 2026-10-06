package calls

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Presence diz se algum navegador do usuário está conectado ao tempo real.
type Presence interface {
	HasSubscribers(room string) bool
}

// ContactResolver acha ou cria o contato do chamador (mesma regra das mensagens,
// inclusive a unificação LID × telefone).
type ContactResolver interface {
	FindOrCreate(ctx context.Context, tenantID uuid.UUID, number, pushName, profilePicURL string, isGroup, isLID bool, from string) (*domain.Contact, error)
}

// Service reúne as regras de negócio das chamadas. Dependências por construtor.
type Service struct {
	db        *gorm.DB
	contacts  ContactResolver
	tickets   domain.TicketRepository
	queues    domain.QueueRepository
	publisher domain.CommandPublisher
	bcast     domain.Broadcaster
	presence  Presence
	now       func() time.Time

	pauseMu sync.RWMutex
	paused  map[string]bool

	qualityMu sync.Mutex
	quality   map[string]*Summary
}

func NewService(db *gorm.DB, contacts ContactResolver, tickets domain.TicketRepository, queues domain.QueueRepository, publisher domain.CommandPublisher, bcast domain.Broadcaster, presence Presence) *Service {
	return &Service{
		db: db, contacts: contacts, tickets: tickets, queues: queues, publisher: publisher,
		bcast: domain.BroadcastOrNop(bcast), presence: presence, now: time.Now,
		paused: map[string]bool{}, quality: map[string]*Summary{},
	}
}

// UserRoom é a sala SSE pessoal do usuário (inscrita pelo servidor a partir do token).
func UserRoom(tenantID uuid.UUID, userID int) string {
	return fmt.Sprintf("user:%s:%d", tenantID, userID)
}

func pauseKey(tenantID uuid.UUID, userID int) string { return fmt.Sprintf("%s:%d", tenantID, userID) }

// SetPaused marca o operador como pausado (ou não) para receber toque. Vive em
// memória: se o processo reinicia, o navegador reenvia o estado ao reconectar.
func (s *Service) SetPaused(tenantID uuid.UUID, userID int, paused bool) {
	s.pauseMu.Lock()
	defer s.pauseMu.Unlock()
	if paused {
		s.paused[pauseKey(tenantID, userID)] = true
		return
	}
	delete(s.paused, pauseKey(tenantID, userID))
}

func (s *Service) isPaused(tenantID uuid.UUID, userID int) bool {
	s.pauseMu.RLock()
	defer s.pauseMu.RUnlock()
	return s.paused[pauseKey(tenantID, userID)]
}

func (s *Service) fresh() *gorm.DB { return s.db.Session(&gorm.Session{NewDB: true}) }

// Eligible devolve os usuários que devem tocar para uma chamada na conexão:
// enxergam a conexão, têm calls:receive, estão online e não pausaram. Sem o
// db do contexto HTTP, a checagem de permissão usa o cadastro do usuário.
func (s *Service) Eligible(tenantID uuid.UUID, whatsappID int, hasPerm func(userID int) bool) ([]models.User, error) {
	candidates, err := UsersWhoSeeConnection(s.db, tenantID, whatsappID)
	if err != nil {
		return nil, err
	}
	var out []models.User
	for _, u := range candidates {
		if u.TenantID != tenantID || !hasPerm(u.ID) || s.isPaused(tenantID, u.ID) {
			continue
		}
		if s.presence == nil || !s.presence.HasSubscribers(UserRoom(tenantID, u.ID)) {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}

// peerNumber extrai só os dígitos de um JID ("5511...@s.whatsapp.net" -> "5511...").
func peerNumber(jid string) string {
	base := strings.Split(jid, "@")[0]
	return strings.Split(base, ":")[0]
}

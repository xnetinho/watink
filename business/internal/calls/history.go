package calls

import (
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HistoryFilter filtra o histórico de chamadas.
type HistoryFilter struct {
	Status    string
	Direction string
	TicketID  int
	Page      int
	PageSize  int
}

// HistoryScope diz o que o usuário pode ver: tudo da empresa (alcance de empresa)
// ou só as chamadas de tickets que ele enxerga.
type HistoryScope struct {
	TenantID uuid.UUID
	UserID   int
	// Tenant é verdadeiro para alcance tenant/plataforma.
	Tenant bool
}

// scoped restringe a consulta à empresa e, sem alcance de empresa, às chamadas
// cujo ticket é visível ao usuário (mesma regra de auth.GetScopedDB("Tickets")).
func (s *Service) scoped(sc HistoryScope) *gorm.DB {
	q := s.fresh().Model(&models.CallLog{}).Where(`"CallLogs"."tenantId" = ?`, sc.TenantID)
	if sc.Tenant {
		return q
	}
	return q.Where(`"CallLogs"."ticketId" IN (SELECT t.id FROM "Tickets" t WHERE t."tenantId" = ? AND (`+
		`t."userId" = ? `+
		`OR t."queueId" IN (SELECT queue_id FROM user_queues WHERE user_id = ?) `+
		`OR t."whatsappId" IN (SELECT wq.whatsapp_id FROM whatsapp_queues wq WHERE wq.queue_id IN (SELECT queue_id FROM user_queues WHERE user_id = ?))))`,
		sc.TenantID, sc.UserID, sc.UserID, sc.UserID)
}

// History lista as chamadas visíveis ao usuário, da mais recente para a mais antiga.
func (s *Service) History(sc HistoryScope, f HistoryFilter) ([]models.CallLog, int64, error) {
	q := s.scoped(sc)
	if f.Status != "" {
		q = q.Where(`"CallLogs".status = ?`, f.Status)
	}
	if f.Direction != "" {
		q = q.Where(`"CallLogs".direction = ?`, f.Direction)
	}
	if f.TicketID > 0 {
		q = q.Where(`"CallLogs"."ticketId" = ?`, f.TicketID)
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	size := f.PageSize
	if size <= 0 || size > 100 {
		size = 20
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	var out []models.CallLog
	err := q.Order(`"CallLogs"."startedAt" DESC, "CallLogs".id DESC`).Limit(size).Offset((page - 1) * size).Find(&out).Error
	return out, total, err
}

// Get devolve uma chamada se o usuário puder vê-la; senão ErrNotFound (nunca
// revela que ela existe).
func (s *Service) Get(sc HistoryScope, callID string) (*models.CallLog, error) {
	var l models.CallLog
	if err := s.scoped(sc).Where(`"CallLogs"."callId" = ?`, callID).First(&l).Error; err != nil {
		return nil, ErrNotFound
	}
	return &l, nil
}

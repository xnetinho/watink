package models

import (
	"time"

	"github.com/google/uuid"
)

// OutboxEvent armazena eventos de domínio pendentes de processamento para garantir
// que nenhum evento se perca caso o processo caia entre a gravação e o tratamento (Tarefa 5.4 / ADR 0033).
type OutboxEvent struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID    uuid.UUID  `gorm:"column:tenantId;type:uuid;not null;index" json:"tenantId"`
	EventName   string     `gorm:"column:eventName;not null;index" json:"eventName"`
	Payload     string     `gorm:"column:payload;type:jsonb;not null" json:"payload"`
	CreatedAt   time.Time  `gorm:"column:createdAt;not null;default:now()" json:"createdAt"`
	ProcessedAt *time.Time `gorm:"column:processedAt;index" json:"processedAt,omitempty"`
}

func (OutboxEvent) TableName() string {
	return "OutboxEvents"
}

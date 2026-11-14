package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tenant struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name    string    `gorm:"not null" json:"name"`
	Status  string    `gorm:"default:'active'" json:"status"`
	OwnerID *int      `gorm:"column:ownerId" json:"ownerId"`
	// DefaultQueueID é a fila que recebe o ticket quando a conexão não tem fila
	// ligada (ou tem mais de uma). Sem ela, só administradores veem o ticket.
	DefaultQueueID *int   `gorm:"column:defaultQueueId" json:"defaultQueueId"`
	Document       string `json:"document"`
	// ProvisionKey é a chave de idempotência do provisionamento pelo control
	// plane SaaS (rota interna POST /internal/saas/tenants). Nullable/único
	// parcial: um retry do outbox com a mesma chave devolve o tenant existente
	// em vez de criar outro. Nunca serializado na API pública (json:"-").
	ProvisionKey *string   `gorm:"column:provisionKey" json:"-"`
	CreatedAt    time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"column:updatedAt" json:"updatedAt"`

	// Relations
	Users []User `gorm:"foreignKey:TenantID" json:"users,omitempty"`
}

func (Tenant) TableName() string {
	return "Tenants"
}

func (t *Tenant) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}

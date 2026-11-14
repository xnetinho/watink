// Package routing decide, num lugar só, a fila de um ticket novo.
//
// Antes cada entrada (mensagem do WhatsApp, canal genérico/Telegram, chamada)
// resolvia a fila do seu jeito — e o canal genérico não resolvia nenhuma.
// A regra é única: a conexão manda quando tem exatamente uma fila; senão vale a
// fila padrão da empresa; sem nenhuma das duas o ticket nasce sem fila.
package routing

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResolveNewTicketQueue devolve a fila em que o ticket novo começa:
//  1. a fila da conexão, quando há exatamente uma ligada a ela;
//  2. senão a fila padrão da empresa (inclusive quando a conexão tem 0 ou mais
//     de uma fila — mais de uma é ambíguo e não se adivinha);
//  3. senão nil: o ticket fica sem fila e só administradores o enxergam.
func ResolveNewTicketQueue(ctx context.Context, db *gorm.DB, tenantID uuid.UUID, connectionID int) *int {
	var ids []int
	err := db.WithContext(ctx).
		Table("channel_connection_queues AS wq").
		Joins(`JOIN "Queues" q ON q.id = wq.queue_id`).
		Where(`wq.channel_connection_id = ? AND q."tenantId" = ?`, connectionID, tenantID).
		Pluck("wq.queue_id", &ids).Error
	if err == nil && len(ids) == 1 {
		return &ids[0]
	}

	var tenant struct {
		DefaultQueueID *int
	}
	if err := db.WithContext(ctx).
		Table(`"Tenants"`).
		Select(`"defaultQueueId"`).
		Where(`id = ?`, tenantID).
		Scan(&tenant).Error; err != nil || tenant.DefaultQueueID == nil {
		return nil
	}

	// A fila padrão pode ter sido excluída antes desta coluna existir: só vale
	// se ainda pertence à empresa.
	var exists int64
	if err := db.WithContext(ctx).
		Table(`"Queues"`).
		Where(`id = ? AND "tenantId" = ?`, *tenant.DefaultQueueID, tenantID).
		Count(&exists).Error; err != nil || exists == 0 {
		return nil
	}
	return tenant.DefaultQueueID
}

package services

import (
	"context"
	"errors"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrQueueNotFound = errors.New("queue not found")
	// ErrQueueMoveInvalid: o destino não existe nesta empresa ou é a própria fila.
	ErrQueueMoveInvalid = errors.New("invalid queue move target")
)

// QueueDependents é o que depende de uma fila e precisa de destino antes de ela sair.
type QueueDependents struct {
	Tickets, Users, Connections, Setores, Children int64
}

func (d QueueDependents) Empty() bool {
	return d.Tickets+d.Users+d.Connections+d.Setores+d.Children == 0
}

// QueueHasDependentsError: a fila ainda tem ticket, usuário, conexão, setor ou fila filha e nenhum destino foi indicado.
type QueueHasDependentsError struct{ Dependents QueueDependents }

func (e *QueueHasDependentsError) Error() string { return "queue has dependents" }

// DeleteQueue exclui a fila. Com dependentes exige o destino (moveTo): tickets, usuários, conexões e setores passam para ele e as filas filhas
// ficam sem pai. Antes o banco recusava a exclusão quando havia usuário ou conexão vinculados, e, sem vínculos, deixava o ticket apontando para uma
// fila que não existe mais. Tudo numa transação: ou move tudo e exclui, ou nada muda.
func DeleteQueue(ctx context.Context, db *gorm.DB, tenantID uuid.UUID, queueID int, moveTo *int) error {
	return db.WithContext(ctx).Session(&gorm.Session{NewDB: true}).Transaction(func(tx *gorm.DB) error {
		var q models.Queue
		if err := tx.Where(`id = ? AND "tenantId" = ?`, queueID, tenantID).First(&q).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrQueueNotFound
			}
			return err
		}
		dep, err := queueDependents(tx, tenantID, queueID)
		if err != nil {
			return err
		}
		if !dep.Empty() {
			if moveTo == nil {
				return &QueueHasDependentsError{Dependents: dep}
			}
			if err := moveQueueDependents(tx, tenantID, queueID, *moveTo); err != nil {
				return err
			}
		}
		return tx.Where(`id = ? AND "tenantId" = ?`, queueID, tenantID).Delete(&models.Queue{}).Error
	})
}

func queueDependents(tx *gorm.DB, tenantID uuid.UUID, queueID int) (QueueDependents, error) {
	var d QueueDependents
	counts := []struct {
		dst *int64
		sql string
		arg []any
	}{
		{&d.Tickets, `SELECT count(*) FROM "Tickets" WHERE "queueId" = ? AND "tenantId" = ?`, []any{queueID, tenantID}},
		{&d.Users, `SELECT count(*) FROM user_queues WHERE queue_id = ?`, []any{queueID}},
		{&d.Connections, `SELECT count(*) FROM channel_connection_queues WHERE queue_id = ?`, []any{queueID}},
		{&d.Setores, `SELECT count(*) FROM setor_filas WHERE "queueId" = ?`, []any{queueID}},
		{&d.Children, `SELECT count(*) FROM "Queues" WHERE "parentId" = ? AND "tenantId" = ?`, []any{queueID, tenantID}},
	}
	for _, c := range counts {
		if err := tx.Raw(c.sql, c.arg...).Scan(c.dst).Error; err != nil {
			return d, err
		}
	}
	return d, nil
}

func moveQueueDependents(tx *gorm.DB, tenantID uuid.UUID, from, to int) error {
	if to == from {
		return ErrQueueMoveInvalid
	}
	var dest models.Queue
	if err := tx.Where(`id = ? AND "tenantId" = ?`, to, tenantID).First(&dest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQueueMoveInvalid
		}
		return err
	}
	steps := []struct {
		sql string
		arg []any
	}{
		{`UPDATE "Tickets" SET "queueId" = ? WHERE "queueId" = ? AND "tenantId" = ?`, []any{to, from, tenantID}},
		{`INSERT INTO user_queues (user_id, queue_id) SELECT user_id, ? FROM user_queues WHERE queue_id = ? ON CONFLICT DO NOTHING`, []any{to, from}},
		{`DELETE FROM user_queues WHERE queue_id = ?`, []any{from}},
		{`INSERT INTO channel_connection_queues (channel_connection_id, queue_id) SELECT channel_connection_id, ? FROM channel_connection_queues WHERE queue_id = ? ON CONFLICT DO NOTHING`, []any{to, from}},
		{`DELETE FROM channel_connection_queues WHERE queue_id = ?`, []any{from}},
		{`INSERT INTO setor_filas ("setorId", "queueId") SELECT "setorId", ? FROM setor_filas WHERE "queueId" = ? ON CONFLICT DO NOTHING`, []any{to, from}},
		{`DELETE FROM setor_filas WHERE "queueId" = ?`, []any{from}},
		{`UPDATE "Queues" SET "parentId" = NULL WHERE "parentId" = ? AND "tenantId" = ?`, []any{from, tenantID}},
	}
	for _, s := range steps {
		if err := tx.Exec(s.sql, s.arg...).Error; err != nil {
			return err
		}
	}
	return nil
}

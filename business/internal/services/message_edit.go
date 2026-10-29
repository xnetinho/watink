package services

import (
	"context"
	"errors"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrEditTargetNotFound: a mensagem editada não existe nesta empresa (ex.: veio antes de a original ser gravada).
var ErrEditTargetNotFound = errors.New("message to edit not found")

// RecordEditInput descreve uma edição. EditedAt é o instante dela segundo o canal: é ele que ordena a cadeia, e não a
// hora em que o evento chegou aqui.
type RecordEditInput struct {
	TenantID  uuid.UUID
	MessageID string
	Body      string
	EditedAt  time.Time
	EditedBy  string // models.EditedByMe | models.EditedByContact
}

// RecordEditResult diz o que aconteceu, para quem chama saber se precisa avisar a tela.
type RecordEditResult struct {
	Message *models.Message
	// Applied: a edição foi gravada e a mensagem mudou. Falso quando é repetição de um evento já visto ou uma edição
	// mais antiga que a atual (fora de ordem), que só entra na cadeia.
	Applied bool
	// Duplicate: já existia uma versão com este mesmo texto e instante. Nada foi gravado.
	Duplicate bool
}

// RecordEdit grava uma edição na cadeia da mensagem e atualiza o texto atual.
//
//   - A primeira edição cria a versão 1 (o texto que a mensagem tinha) e a 2 (o texto novo).
//   - As seguintes acrescentam a próxima versão.
//   - O mesmo evento duas vezes (o engine entrega ao menos uma vez) não duplica nada.
//   - Edição fora de ordem entra na cadeia, mas o texto atual da mensagem é o da edição de maior EditedAt.
//
// Tudo numa transação com a mensagem travada (FOR UPDATE): duas edições simultâneas da mesma mensagem não disputam a
// mesma versão. O banco ainda garante (messageId, version) único.
//
// Vale para qualquer canal: o WhatsApp e o Telegram chegam aqui pelo mesmo caminho.
func RecordEdit(ctx context.Context, db *gorm.DB, in RecordEditInput) (RecordEditResult, error) {
	var res RecordEditResult
	if in.EditedAt.IsZero() {
		in.EditedAt = time.Now()
	}
	if in.EditedBy == "" {
		in.EditedBy = models.EditedByContact
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var msg models.Message
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(`id = ? AND "tenantId" = ?`, in.MessageID, in.TenantID).First(&msg).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEditTargetNotFound
			}
			return err
		}

		var versions []models.MessageEdit
		if err := tx.Where(`"messageId" = ?`, msg.ID).Order("version ASC").Find(&versions).Error; err != nil {
			return err
		}

		// Repetição do mesmo evento: mesma hora de edição e mesmo texto.
		for _, v := range versions {
			if v.EditedAt.Equal(in.EditedAt) && v.Body == in.Body {
				res.Message, res.Duplicate = &msg, true
				return nil
			}
		}

		// Primeira edição: o texto que a mensagem tinha até agora é a versão 1.
		if len(versions) == 0 {
			original := models.MessageEdit{
				TenantID: in.TenantID, MessageID: msg.ID, Version: 1, Body: msg.Body,
				EditedAt: msg.CreatedAt, EditedBy: originalAuthor(msg), CreatedAt: time.Now(),
			}
			if err := tx.Create(&original).Error; err != nil {
				return err
			}
			versions = append(versions, original)
		}

		next := models.MessageEdit{
			TenantID: in.TenantID, MessageID: msg.ID, Version: versions[len(versions)-1].Version + 1,
			Body: in.Body, EditedAt: in.EditedAt, EditedBy: in.EditedBy, CreatedAt: time.Now(),
		}
		if err := tx.Create(&next).Error; err != nil {
			return err
		}

		// O texto atual é o da versão mais recente por EditedAt (a ordem de chegada pode embaralhar).
		current := next
		for _, v := range versions {
			if v.Version > 1 && v.EditedAt.After(current.EditedAt) {
				current = v
			}
		}
		res.Applied = current.Version == next.Version
		msg.EditCount = next.Version - 1
		upd := map[string]interface{}{"editCount": msg.EditCount, "updatedAt": time.Now()}
		if res.Applied {
			upd["body"] = in.Body
			msg.Body = in.Body
		} else {
			msg.Body = current.Body
		}
		if err := tx.Model(&models.Message{}).Where(`id = ? AND "tenantId" = ?`, msg.ID, in.TenantID).Updates(upd).Error; err != nil {
			return err
		}
		res.Message = &msg
		return nil
	})
	return res, err
}

// originalAuthor: quem escreveu a versão 1. Mensagem nossa foi do operador; a do contato, do contato.
func originalAuthor(m models.Message) string {
	if m.FromMe {
		return models.EditedByMe
	}
	return models.EditedByContact
}

// MessageEdits devolve a cadeia da mensagem, da versão 1 (original) à mais recente, só da empresa informada.
func MessageEdits(ctx context.Context, db *gorm.DB, tenantID uuid.UUID, messageID string) ([]models.MessageEdit, error) {
	var out []models.MessageEdit
	err := db.WithContext(ctx).
		Where(`"messageId" = ? AND "tenantId" = ?`, messageID, tenantID).
		Order("version ASC").Find(&out).Error
	return out, err
}

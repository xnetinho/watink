package services

import (
	"context"
	"errors"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound = errors.New("user not found")
	// ErrWalletTargetInvalid: quem recebe a carteira não existe nesta empresa ou é o próprio usuário excluído.
	ErrWalletTargetInvalid = errors.New("invalid wallet target")
)

// UserWalletError: o usuário é responsável por contatos e nenhum destino para a carteira foi indicado.
type UserWalletError struct{ Contacts int64 }

func (e *UserWalletError) Error() string { return "user has a contact wallet" }

// DeleteUser exclui o usuário sem deixar nada preso nem pendurado, numa transação:
//   - carteira de contatos: vai para walletTo (obrigatório se houver contatos);
//   - tickets abertos ou pendentes (fora de grupo): perdem o dono e voltam a pendente, na fila deles, para outro aceitar; os de grupo e os fechados só perdem o dono;
//   - vínculos de fila e de setor, e o histórico que o aponta (log de ticket, de protocolo, acesso a gravação, responsável de atividade): removidos.
//
// Antes o banco recusava a exclusão sempre que havia ticket, carteira, vínculo de fila ou log do usuário.
// shortcut: os tickets soltos voltam à fila sem redistribuição automática; a distribuição passa a atuar neles com o ponto único de roteamento (PR 2 do desenho da fila).
func DeleteUser(ctx context.Context, db *gorm.DB, tenantID uuid.UUID, userID int, walletTo *int) error {
	return db.WithContext(ctx).Session(&gorm.Session{NewDB: true}).Transaction(func(tx *gorm.DB) error {
		var u models.User
		if err := tx.Where(`id = ? AND "tenantId" = ?`, userID, tenantID).First(&u).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		var contacts int64
		if err := tx.Raw(`SELECT count(*) FROM "Contacts" WHERE "walletUserId" = ? AND "tenantId" = ?`, userID, tenantID).Scan(&contacts).Error; err != nil {
			return err
		}
		if contacts > 0 {
			if walletTo == nil {
				return &UserWalletError{Contacts: contacts}
			}
			if *walletTo == userID {
				return ErrWalletTargetInvalid
			}
			var dest models.User
			if err := tx.Where(`id = ? AND "tenantId" = ?`, *walletTo, tenantID).First(&dest).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrWalletTargetInvalid
				}
				return err
			}
			if err := tx.Exec(`UPDATE "Contacts" SET "walletUserId" = ? WHERE "walletUserId" = ? AND "tenantId" = ?`, *walletTo, userID, tenantID).Error; err != nil {
				return err
			}
		}
		steps := []struct {
			sql string
			arg []any
		}{
			{`UPDATE "Tickets" SET "userId" = NULL, status = 'pending', "updatedAt" = now() WHERE "userId" = ? AND "tenantId" = ? AND status IN ('open','pending') AND "isGroup" = false`, []any{userID, tenantID}},
			{`UPDATE "Tickets" SET "userId" = NULL WHERE "userId" = ? AND "tenantId" = ?`, []any{userID, tenantID}},
			{`DELETE FROM "TicketLogs" WHERE "userId" = ? AND "tenantId" = ?`, []any{userID, tenantID}},
			{`DELETE FROM "ProtocolLogs" WHERE "userId" = ? AND "tenantId" = ?`, []any{userID, tenantID}},
			{`DELETE FROM "CallRecordingAccesses" WHERE "userId" = ? AND "tenantId" = ?`, []any{userID, tenantID}},
			{`DELETE FROM "ActivityAssignees" WHERE "userId" = ? AND "tenantId" = ?`, []any{userID, tenantID}},
			{`UPDATE "CallLogs" SET "handledByUserId" = NULL WHERE "handledByUserId" = ? AND "tenantId" = ?`, []any{userID, tenantID}},
			{`DELETE FROM user_queues WHERE user_id = ?`, []any{userID}},
			{`DELETE FROM user_setores WHERE "userId" = ?`, []any{userID}},
		}
		for _, s := range steps {
			if err := tx.Exec(s.sql, s.arg...).Error; err != nil {
				return err
			}
		}
		return tx.Where(`id = ? AND "tenantId" = ?`, userID, tenantID).Delete(&models.User{}).Error
	})
}

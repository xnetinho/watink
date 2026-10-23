package calls

import (
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsersWhoSeeConnection devolve os usuários da empresa que enxergam a conexão
// (WhatsApp) `whatsappID`: alcance de empresa/plataforma, ou a conexão está
// ligada a uma fila do usuário (whatsapp_queues × user_queues).
//
// Espelha a regra de visibilidade de tickets de auth.GetScopedDB("Tickets"), que
// só existe como SQL dentro de um middleware HTTP. O teste de paridade
// (visibility_test.go) compara as duas e acusa se algum dia divergirem.
//
// User.WhatsappID NÃO entra: é só a conexão preferida ao criar um ticket
// (ticket_create.go) e não dá visibilidade de ticket. Tocar o telefone de quem
// depois não consegue abrir o ticket da chamada seria pior que não tocar.
//
// A conexão tem de ser da empresa: um whatsappID de outra empresa devolve vazio.
func UsersWhoSeeConnection(db *gorm.DB, tenantID uuid.UUID, whatsappID int) ([]models.User, error) {
	fresh := func() *gorm.DB { return db.Session(&gorm.Session{NewDB: true}) }

	var owned int64
	if err := fresh().Model(&models.Whatsapp{}).
		Where(`id = ? AND "tenantId" = ?`, whatsappID, tenantID).Count(&owned).Error; err != nil {
		return nil, err
	}
	if owned == 0 {
		return nil, nil
	}

	var users []models.User
	err := fresh().Where(
		`"tenantId" = ? AND ( alcance IN ('tenant','plataforma') `+
			`OR id IN (SELECT uq.user_id FROM user_queues uq `+
			`JOIN whatsapp_queues wq ON wq.queue_id = uq.queue_id WHERE wq.whatsapp_id = ?) )`,
		tenantID, whatsappID).
		Find(&users).Error
	return users, err
}

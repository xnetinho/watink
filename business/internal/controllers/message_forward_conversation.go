package controllers

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
)

var errNoConversationKey = errors.New("contato sem identificação na conexão")

// conversationFor devolve o ticket aberto/pendente do contato NA CONEXÃO DELE, criando-o quando não existe. É o que
// permite encaminhar para um contato que nunca conversou: a conversa nasce já ligada à conexão do contato.
func (mc *MessageController) conversationFor(db *gorm.DB, tenantID uuid.UUID, contact models.Contact, conn resolvedConnection) (models.Ticket, error) {
	q := func() *gorm.DB { return db.Session(&gorm.Session{NewDB: true}) }
	var t models.Ticket

	// A chave da conversa depende do canal: nos agnósticos (Telegram) é a do canal, guardada na identidade do contato; no WhatsApp
	// é a do próprio contato, que já unifica telefone e LID. Daí em diante o caminho é o mesmo.
	convKey := domain.WhatsAppConversationKey(contact.ID)
	if conn.Kind != string(domain.ChannelKindWhatsApp) {
		var ident models.ContactIdentity
		if err := q().Where(`"tenantId" = ? AND "contactId" = ? AND "connectionId" = ?`, tenantID, contact.ID, conn.ID).First(&ident).Error; err != nil {
			return t, errNoConversationKey
		}
		convKey = ident.ExternalID
	}
	err := q().Preload("Contact").Where(`"tenantId" = ? AND "channelConnectionId" = ? AND "conversationKey" = ? AND status IN ?`,
		tenantID, conn.ID, convKey, []string{"open", "pending"}).First(&t).Error
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return t, err
	}
	t = models.Ticket{TenantID: tenantID, ContactID: contact.ID, ChannelConnectionID: conn.ID, ConversationKey: convKey, Status: "open", IsGroup: contact.IsGroup}
	if err := q().Create(&t).Error; err != nil {
		return t, err
	}
	t.Contact = contact
	return t, nil
}

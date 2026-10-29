package controllers

import (
	"errors"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var errNoConversationKey = errors.New("contato sem identificação na conexão")

// conversationFor devolve o ticket aberto/pendente do contato NA CONEXÃO DELE, criando-o quando não existe. É o que
// permite encaminhar para um contato que nunca conversou: a conversa nasce já ligada à conexão do contato.
func (mc *MessageController) conversationFor(db *gorm.DB, tenantID uuid.UUID, contact models.Contact, conn resolvedConnection) (models.Ticket, error) {
	q := func() *gorm.DB { return db.Session(&gorm.Session{NewDB: true}) }
	var t models.Ticket

	if conn.Ref.Source == domain.ConnectionSourceChannel {
		// Canais agnósticos: a conversa é identificada pela chave do canal (Telegram: chat_id), guardada na identidade.
		var ident models.ContactIdentity
		if err := q().Where(`"tenantId" = ? AND "contactId" = ? AND "connectionId" = ?`, tenantID, contact.ID, conn.Ref.ID).First(&ident).Error; err != nil {
			return t, errNoConversationKey
		}
		err := q().Preload("Contact").Where(`"tenantId" = ? AND "channelConnectionId" = ? AND "conversationKey" = ? AND status IN ?`,
			tenantID, conn.Ref.ID, ident.ExternalID, []string{"open", "pending"}).First(&t).Error
		if err == nil {
			return t, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return t, err
		}
		connID := conn.Ref.ID
		t = models.Ticket{TenantID: tenantID, ContactID: contact.ID, ChannelConnectionID: &connID, ConversationKey: ident.ExternalID, Status: "open", IsGroup: contact.IsGroup}
		if err := q().Create(&t).Error; err != nil {
			return t, err
		}
		t.Contact = contact
		return t, nil
	}

	err := q().Preload("Contact").Where(`"tenantId" = ? AND "contactId" = ? AND "whatsappId" = ? AND status IN ?`,
		tenantID, contact.ID, conn.Ref.ID, []string{"open", "pending"}).First(&t).Error
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return t, err
	}
	t = models.Ticket{TenantID: tenantID, ContactID: contact.ID, WhatsappID: conn.Ref.ID, Status: "open", IsGroup: contact.IsGroup}
	if err := q().Create(&t).Error; err != nil {
		return t, err
	}
	t.Contact = contact
	return t, nil
}

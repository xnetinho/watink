package controllers

import (
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// connectionIDOf é o id da conexão do ticket em ChannelConnections. O ticket do WhatsApp o guarda em whatsappId (a migração
// leva tudo para channelConnectionId na fatia 4); os demais canais, em channelConnectionId.
func connectionIDOf(t models.Ticket) int {
	if t.ChannelConnectionID != nil {
		return *t.ChannelConnectionID
	}
	return t.WhatsappID
}

// channelOf resolve a conexão do ticket e o adaptador do seu canal. found=false: a conexão não existe nesta empresa.
func channelOf(db *gorm.DB, tenantID uuid.UUID, t models.Ticket) (conn models.ChannelConnection, ch domain.Channel, found bool, err error) {
	if err := db.Session(&gorm.Session{NewDB: true}).
		Where(`id = ? AND "tenantId" = ?`, connectionIDOf(t), tenantID).First(&conn).Error; err != nil {
		return conn, nil, false, nil
	}
	ch, err = domain.DefaultChannelRegistry().Get(domain.ChannelKind(conn.Kind))
	return conn, ch, true, err
}

// channelOfConnection é channelOf para quem já tem o id da conexão (contatos), não um ticket.
func channelOfConnection(db *gorm.DB, tenantID uuid.UUID, connID int) (conn models.ChannelConnection, ch domain.Channel, found bool, err error) {
	return channelOf(db, tenantID, models.Ticket{WhatsappID: connID})
}

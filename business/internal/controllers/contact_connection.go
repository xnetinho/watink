package controllers

import (
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// resolvedConnection é uma conexão da empresa, já validada.
type resolvedConnection struct {
	Ref    domain.ConnectionRef
	Kind   string // "whatsapp", "telegram"...
	Name   string
	Status string
}

// resolveConnection acha a conexão pelo par (origem, id) DENTRO da empresa. Fonte única da regra "esta conexão existe
// e é desta empresa": criar contato, importar e encaminhar passam por aqui, então nenhum aceita a conexão de outra
// empresa nem uma conexão inexistente.
func resolveConnection(db *gorm.DB, tenantID uuid.UUID, source string, id int) (resolvedConnection, bool) {
	s := db.Session(&gorm.Session{NewDB: true})
	switch source {
	case domain.ConnectionSourceWhatsapp, "":
		var w models.Whatsapp
		if s.Select("id", "name", "status").Where(`id = ? AND "tenantId" = ?`, id, tenantID).First(&w).Error != nil {
			return resolvedConnection{}, false
		}
		return resolvedConnection{Ref: domain.WhatsappConnection(w.ID), Kind: string(domain.ChannelKindWhatsApp), Name: w.Name, Status: w.Status}, true
	case domain.ConnectionSourceChannel:
		var c models.ChannelConnection
		if s.Select("id", "kind", "name", "status").Where(`id = ? AND "tenantId" = ?`, id, tenantID).First(&c).Error != nil {
			return resolvedConnection{}, false
		}
		return resolvedConnection{Ref: domain.ChannelConnection(c.ID), Kind: c.Kind, Name: c.Name, Status: c.Status}, true
	}
	return resolvedConnection{}, false
}

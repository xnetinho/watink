package controllers

import (
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// resolvedConnection é uma conexão da empresa, já validada.
type resolvedConnection struct {
	ID     int
	Kind   string // "whatsapp", "telegram"...
	Name   string
	Status string
}

// resolveConnection acha a conexão pelo id DENTRO da empresa. Fonte única da regra "esta conexão existe e é desta empresa":
// criar contato, importar e encaminhar passam por aqui, então nenhum aceita a conexão de outra empresa nem uma inexistente.
func resolveConnection(db *gorm.DB, tenantID uuid.UUID, id int) (resolvedConnection, bool) {
	var c models.ChannelConnection
	if db.Session(&gorm.Session{NewDB: true}).Select("id", "kind", "name", "status").Where(`id = ? AND "tenantId" = ?`, id, tenantID).First(&c).Error != nil {
		return resolvedConnection{}, false
	}
	return resolvedConnection{ID: c.ID, Kind: c.Kind, Name: c.Name, Status: c.Status}, true
}

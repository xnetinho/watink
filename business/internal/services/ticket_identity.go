package services

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
)

// withConnectionIdentity devolve o ticket como mapa JSON com "channel" e "connection" (canal + conexão), para o evento em
// tempo real já trazer o que a lista precisa para identificar o ticket. O ticket que nasce por mensagem só chega à tela
// por este evento, então sem isso ele apareceria sem identificação até recarregar.
//
// Falha silenciosa por desenho: sem a identificação a tela só mostra o canal, e o recebimento da mensagem nunca depende
// dela. O ctx PRECISA carregar o escopo da empresa (RLS, ADR 0034): sem ele a consulta é recusada com ErrNoScope e o
// ticket sai sem identificação (docs/dev/rls-escopo-de-empresa.md).
// WithConnectionIdentity é o ponto exportado de withConnectionIdentity, para os testes de integração com RLS real.
func WithConnectionIdentity(ctx context.Context, db *gorm.DB, ticket *domain.Ticket) interface{} {
	return withConnectionIdentity(ctx, db, ticket)
}

func withConnectionIdentity(ctx context.Context, db *gorm.DB, ticket *domain.Ticket) interface{} {
	if db == nil || ticket == nil {
		return ticket
	}
	raw, err := json.Marshal(ticket)
	if err != nil {
		return ticket
	}
	var out map[string]interface{}
	if json.Unmarshal(raw, &out) != nil {
		return ticket
	}
	out["channel"] = string(domain.ChannelKindWhatsApp)
	if c, ok := ConnectionOf(db.WithContext(ctx), ticket.ChannelConnectionID); ok {
		out["channel"] = c.Kind
		out["connection"] = c
	}
	return out
}

// ConnectionOf resolve a conexão de UM ticket (canal + nome) em ChannelConnections. Com o RLS ativo, a conexão de outra empresa
// nunca é encontrada, então o nome dela não vaza.
func ConnectionOf(db *gorm.DB, connectionID int) (models.TicketConnectionInfo, bool) {
	var c models.ChannelConnection
	if err := db.Session(&gorm.Session{NewDB: true}).Select("id", "kind", "name", "avatarUrl").Where("id = ?", connectionID).First(&c).Error; err != nil {
		return models.TicketConnectionInfo{}, false
	}
	return models.TicketConnectionInfo{Kind: c.Kind, ID: c.ID, Name: c.Name, Source: models.ConnectionSource(c.Kind), ProfilePicUrl: c.AvatarURL}, true
}

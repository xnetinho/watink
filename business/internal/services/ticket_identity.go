package services

import (
	"context"
	"encoding/json"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"gorm.io/gorm"
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
	if c, ok := ConnectionOf(db.WithContext(ctx), ticket.ID, ticket.WhatsappID, ticket.ChannelConnectionID); ok {
		out["channel"] = c.Kind
		out["connection"] = c
	}
	return out
}

// ConnectionOf resolve a conexão de UM ticket (canal + nome). Fonte única da regra "de qual tabela vem a conexão":
// ChannelConnectionID manda; sem ele, o ticket é do WhatsApp e a conexão é a linha de Whatsapps.
func ConnectionOf(db *gorm.DB, ticketID, whatsappID int, channelConnectionID *int) (models.TicketConnectionInfo, bool) {
	s := db.Session(&gorm.Session{NewDB: true})
	if channelConnectionID != nil {
		var c models.ChannelConnection
		if err := s.Select("id", "kind", "name").Where("id = ?", *channelConnectionID).First(&c).Error; err != nil {
			return models.TicketConnectionInfo{}, false
		}
		return models.TicketConnectionInfo{Kind: c.Kind, ID: c.ID, Name: c.Name, Source: "channel"}, true
	}
	if whatsappID == 0 {
		return models.TicketConnectionInfo{}, false
	}
	var w models.Whatsapp
	if err := s.Select("id", "name").Where("id = ?", whatsappID).First(&w).Error; err != nil {
		return models.TicketConnectionInfo{}, false
	}
	return models.TicketConnectionInfo{Kind: string(domain.ChannelKindWhatsApp), ID: w.ID, Name: w.Name, Source: "whatsapp"}, true
}

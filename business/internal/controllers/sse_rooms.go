package controllers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// knownTicketStatuses são os únicos valores que o frontend usa em `tickets:<status>`.
var knownTicketStatuses = map[string]bool{"open": true, "pending": true, "closed": true}

// userRoom é a sala de um usuário específico. O servidor a inscreve a partir do
// token; o cliente nunca a pede por query (allowedExtraRooms a descarta).
func userRoom(tenantID uuid.UUID, userID int) string {
	return fmt.Sprintf("user:%s:%d", tenantID, userID)
}

// allowedExtraRooms filtra o `?rooms=` do SSE. Antes dele, qualquer valor era
// aceito, então um usuário da empresa A pedindo `tenant:<id da B>` recebia os
// eventos da B. Só passam:
//
//	chat:<ticketId>        se canSeeTicket(ticketId) — mesma visibilidade de tickets
//	tickets:<status>       open | pending | closed
//	helpdesk-kanban
//	notification           (já é inscrita por padrão; aceita por compatibilidade)
//
// Todo o resto (tenant:*, user:*, qualquer lixo) é descartado em silêncio.
func allowedExtraRooms(raw string, tenantID uuid.UUID, userID int, canSeeTicket func(ticketID int) bool) []string {
	_ = tenantID
	_ = userID
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	add := func(room string) {
		if !seen[room] {
			seen[room] = true
			out = append(out, room)
		}
	}
	for _, r := range strings.Split(raw, ",") {
		r = strings.TrimSpace(r)
		switch {
		case r == "helpdesk-kanban" || r == "notification":
			add(r)
		case strings.HasPrefix(r, "chat:"):
			id, err := strconv.Atoi(strings.TrimPrefix(r, "chat:"))
			if err == nil && id > 0 && canSeeTicket != nil && canSeeTicket(id) {
				add("chat:" + strconv.Itoa(id))
			}
		case strings.HasPrefix(r, "tickets:"):
			if status := strings.TrimPrefix(r, "tickets:"); knownTicketStatuses[status] {
				add(r)
			}
		}
	}
	return out
}

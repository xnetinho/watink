package controllers

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func recvd(ch <-chan string) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// Reproduz o vazamento que existia: o usuário da empresa A pedia, por query,
// a sala da empresa B e recebia os eventos dela.
func TestSSERooms_ForeignTenantRoomIsDropped(t *testing.T) {
	hub := services.NewSSEHub()
	bc := services.NewSSEBroadcast(hub)
	a, b := uuid.New(), uuid.New()

	rooms := allowedExtraRooms("tenant:"+b.String(), a, 7, func(int) bool { return true })
	ch, clean := hub.Register("conn", append([]string{"tenant:" + a.String(), "notification"}, rooms...))
	defer clean()

	bc.EmitToTenantRoom(b.String(), "call.incoming", map[string]string{"x": "segredo da B"})
	assert.False(t, recvd(ch), "sala de outra empresa nunca pode ser assinada via ?rooms=")

	bc.EmitToTenantRoom(a.String(), "ticket", map[string]string{"x": "da A"})
	assert.True(t, recvd(ch), "a própria sala segue funcionando")
}

func TestAllowedExtraRooms_Whitelist(t *testing.T) {
	tenant := uuid.New()
	canSee := func(id int) bool { return id == 10 }
	got := allowedExtraRooms(
		" chat:10 , chat:99 , tickets:open , tickets:pending , helpdesk-kanban , notification , "+
			"tenant:"+uuid.NewString()+" , tenant:"+tenant.String()+" , user:"+tenant.String()+":7 , x , chat:abc , chat: , tickets: , ,",
		tenant, 7, canSee)

	assert.ElementsMatch(t, []string{"chat:10", "tickets:open", "tickets:pending", "helpdesk-kanban", "notification"}, got,
		"só salas legítimas: chat de ticket visível, tickets:<status>, kanban; nunca tenant:*, user:* nem lixo")
}

func TestAllowedExtraRooms_ChatOfInvisibleTicketIsDropped(t *testing.T) {
	got := allowedExtraRooms("chat:5,chat:6", uuid.New(), 1, func(id int) bool { return id == 6 })
	assert.Equal(t, []string{"chat:6"}, got)
}

func TestAllowedExtraRooms_EmptyAndDuplicates(t *testing.T) {
	assert.Empty(t, allowedExtraRooms("", uuid.New(), 1, func(int) bool { return true }))
	got := allowedExtraRooms("helpdesk-kanban,helpdesk-kanban,chat:1,chat:1", uuid.New(), 1, func(int) bool { return true })
	assert.Equal(t, []string{"helpdesk-kanban", "chat:1"}, got, "sem duplicatas")
}

func TestAllowedExtraRooms_StatusMustBeKnown(t *testing.T) {
	got := allowedExtraRooms("tickets:open,tickets:closed,tickets:pending,tickets:../../etc,tickets:%00", uuid.New(), 1, func(int) bool { return true })
	assert.ElementsMatch(t, []string{"tickets:open", "tickets:closed", "tickets:pending"}, got)
}

func TestUserRoomName(t *testing.T) {
	tenant := uuid.New()
	assert.Equal(t, "user:"+tenant.String()+":42", userRoom(tenant, 42))
}

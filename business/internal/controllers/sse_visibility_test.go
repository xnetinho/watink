package controllers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
)

// Visibilidade Q4 (PR 2b) também no SSE: o operador não assina o chat de um
// ticket que é de outro operador, mesmo sendo da mesma fila.
func TestSSECanSeeTicket_SegueVisibilidadeQ4(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()

	me := models.User{Name: "eu", Email: "eu@test.com", PasswordHash: "h", TenantID: tenant}
	colleague := models.User{Name: "colega", Email: "colega@test.com", PasswordHash: "h", TenantID: tenant}
	require.NoError(t, db.Create(&me).Error)
	require.NoError(t, db.Create(&colleague).Error)

	queue := models.Queue{Name: "minha", Color: "#000", TenantID: tenant}
	otherQueue := models.Queue{Name: "outra", Color: "#111", TenantID: tenant}
	require.NoError(t, db.Create(&queue).Error)
	require.NoError(t, db.Create(&otherQueue).Error)
	require.NoError(t, db.Exec(`INSERT INTO user_queues (user_id, queue_id) VALUES (?,?)`, me.ID, queue.ID).Error)
	require.NoError(t, db.Exec(`INSERT INTO channel_connection_queues (channel_connection_id, queue_id) VALUES (?,?)`, 4, queue.ID).Error)

	mine := visTicket(t, db, tenant, 4, &queue.ID, &me.ID)
	unassigned := visTicket(t, db, tenant, 4, &queue.ID, nil)
	colleagues := visTicket(t, db, tenant, 4, &queue.ID, &colleague.ID)
	otherQueueTicket := visTicket(t, db, tenant, 9, &otherQueue.ID, nil)

	sc := NewSSEController(nil, nil, db)
	see := sc.canSeeTicket(tenant, me.ID, "proprio")
	assert.True(t, see(mine), "ticket próprio")
	assert.True(t, see(unassigned), "ticket sem dono na minha fila")
	assert.False(t, see(colleagues), "ticket de outro operador na mesma fila")
	assert.False(t, see(otherQueueTicket), "ticket sem dono de outra fila")

	admin := sc.canSeeTicket(tenant, me.ID, "tenant")
	assert.True(t, admin(colleagues), "alcance de empresa vê todos")
}

func visTicket(t *testing.T, db *gorm.DB, tenant uuid.UUID, connID int, queueID, userID *int) int {
	t.Helper()
	contact := models.Contact{Name: "c", Number: uuid.NewString()[:8], ConnectionID: connID, TenantID: tenant}
	require.NoError(t, db.Create(&contact).Error)
	ticket := models.Ticket{TenantID: tenant, ContactID: contact.ID, ChannelConnectionID: connID, Status: "pending", QueueID: queueID, UserID: userID}
	require.NoError(t, db.Create(&ticket).Error)
	return ticket.ID
}

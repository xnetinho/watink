package calls

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
)

// Visibilidade Q4 (PR 2b) no histórico de chamadas: a chamada de um ticket de
// outro operador não aparece para quem não tem alcance de empresa.
func TestHistory_OperadorNaoVeChamadaDeTicketDeOutroOperador(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()

	me := models.User{Name: "eu", Email: "eu@test.com", PasswordHash: "h", TenantID: tenant}
	colleague := models.User{Name: "colega", Email: "colega@test.com", PasswordHash: "h", TenantID: tenant}
	require.NoError(t, db.Create(&me).Error)
	require.NoError(t, db.Create(&colleague).Error)

	connID := 5
	mine := ticketWithOwner(t, db, tenant, connID, &me.ID)
	theirs := ticketWithOwner(t, db, tenant, connID, &colleague.ID)

	require.NoError(t, db.Create(&models.CallLog{TenantID: tenant, CallID: "c1", ChannelConnectionID: connID, TicketID: &mine, Direction: "incoming", Status: "ended"}).Error)
	require.NoError(t, db.Create(&models.CallLog{TenantID: tenant, CallID: "c2", ChannelConnectionID: connID, TicketID: &theirs, Direction: "incoming", Status: "ended"}).Error)

	svc := &Service{db: db}
	logs, total, err := svc.History(context.Background(), HistoryScope{TenantID: tenant, UserID: me.ID}, HistoryFilter{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, logs, 1)
	assert.Equal(t, "c1", logs[0].CallID)
}

func TestHistory_AlcanceDeEmpresaVeAsDuas(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	me := models.User{Name: "eu", Email: "eu@test.com", PasswordHash: "h", TenantID: tenant}
	require.NoError(t, db.Create(&me).Error)
	connID := 6
	mine := ticketWithOwner(t, db, tenant, connID, &me.ID)
	unassigned := ticketWithOwner(t, db, tenant, connID, nil)

	require.NoError(t, db.Create(&models.CallLog{TenantID: tenant, CallID: "d1", ChannelConnectionID: connID, TicketID: &mine, Direction: "incoming", Status: "ended"}).Error)
	require.NoError(t, db.Create(&models.CallLog{TenantID: tenant, CallID: "d2", ChannelConnectionID: connID, TicketID: &unassigned, Direction: "incoming", Status: "ended"}).Error)

	svc := &Service{db: db}
	_, total, err := svc.History(context.Background(), HistoryScope{TenantID: tenant, UserID: me.ID, Tenant: true}, HistoryFilter{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
}

func ticketWithOwner(t *testing.T, db *gorm.DB, tenant uuid.UUID, connID int, userID *int) int {
	t.Helper()
	contact := models.Contact{Name: "c", Number: uuid.NewString()[:8], ConnectionID: connID, TenantID: tenant}
	require.NoError(t, db.Create(&contact).Error)
	ticket := models.Ticket{TenantID: tenant, ContactID: contact.ID, ChannelConnectionID: connID, Status: "pending", UserID: userID}
	require.NoError(t, db.Create(&ticket).Error)
	return ticket.ID
}

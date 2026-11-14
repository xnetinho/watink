package database

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
)

// Visibilidade Q4 (PR 2b): operador vê os tickets dele + os SEM DONO das filas
// dele; nunca os de outro operador. Banco migrado de verdade + RLS no papel do app.

type visWorld struct {
	env                  *qEnv
	me, colleague        int
	myQueue, otherQueue  int
	myConn, otherConn    int
	myTicket, queueOpen  int
	channelOpen          int
	colleagueTicket      int
	otherQueueTicket     int
	colleagueOtherTicket int
	walletContact        int
}

func newVisWorld(t *testing.T) *visWorld {
	t.Helper()
	e := newQEnv(t)
	w := &visWorld{env: e}
	w.me = e.user(t, "me@test.com")
	w.colleague = e.user(t, "colleague@test.com")
	w.myQueue = e.queue(t, "minha")
	w.otherQueue = e.queue(t, "outra")
	e.addToQueue(t, w.me, w.myQueue)
	w.myConn = e.conn(t, "meu")
	w.otherConn = e.conn(t, "outro")
	e.linkQueue(t, w.myConn, w.myQueue)

	w.myTicket = e.ticket(t, "wa:1", w.myConn, &w.myQueue, &w.me, "open", false)
	w.queueOpen = e.ticket(t, "wa:2", w.myConn, &w.myQueue, nil, "pending", false)
	w.channelOpen = e.ticket(t, "wa:3", w.myConn, nil, nil, "pending", false)
	w.colleagueTicket = e.ticket(t, "wa:4", w.myConn, &w.myQueue, &w.colleague, "open", false)
	w.otherQueueTicket = e.ticket(t, "wa:5", w.otherConn, &w.otherQueue, nil, "pending", false)
	w.colleagueOtherTicket = e.ticket(t, "wa:6", w.otherConn, &w.otherQueue, &w.colleague, "open", false)

	w.walletContact = e.one(t, `INSERT INTO "Contacts" (name, number, "tenantId", "connectionId", "walletUserId") VALUES ('carteira','wa:9',$1,$2,$3) RETURNING id`, e.tenant, w.myConn, w.me)
	return w
}

func (w *visWorld) visible(t *testing.T, table string, alcance string, userID int, dest any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", w.env.gdb)
	c.Set("tenantId", w.env.tenant)
	c.Set("alcance", alcance)
	c.Set("userId", float64(userID))

	require.NoError(t, tenantdb.WithTenant(context.Background(), w.env.gdb, w.env.tenant, func(ctx context.Context) error {
		return auth.GetScopedDB(c, table).WithContext(ctx).Table(`"`+table+`"`).Order("id").Pluck("id", dest).Error
	}))
}

func TestTicketVisibility_OperadorVeOsDeleESemDonoDasFilasDele(t *testing.T) {
	w := newVisWorld(t)
	var ids []int
	w.visible(t, "Tickets", "proprio", w.me, &ids)

	assert.Equal(t, []int{w.myTicket, w.queueOpen, w.channelOpen}, ids,
		"deve ver o próprio, os sem dono da fila e os sem dono da conexão; nunca o do colega")
}

func TestTicketVisibility_AlcanceDeEmpresaVeTudo(t *testing.T) {
	w := newVisWorld(t)
	var ids []int
	w.visible(t, "Tickets", "tenant", w.me, &ids)

	assert.Equal(t, []int{
		w.myTicket, w.queueOpen, w.channelOpen, w.colleagueTicket, w.otherQueueTicket, w.colleagueOtherTicket,
	}, ids)
}

func TestContactsVisibility_SegueOsTicketsVisiveis(t *testing.T) {
	w := newVisWorld(t)
	var ids []int
	w.visible(t, "Contacts", "proprio", w.me, &ids)

	// Carteira do usuário + contatos dos tickets visíveis (1,2,3); o contato do
	// ticket do colega (4) fica de fora.
	assert.Len(t, ids, 4)
	assert.Contains(t, ids, w.walletContact)
}

func TestContactsVisibility_NaoVazaContatoDeTicketDeOutroOperador(t *testing.T) {
	w := newVisWorld(t)
	var ids []int
	w.visible(t, "Contacts", "proprio", w.me, &ids)

	var colleagueContact int
	require.NoError(t, w.env.admin.QueryRow(`SELECT "contactId" FROM "Tickets" WHERE id = $1`, w.colleagueTicket).Scan(&colleagueContact))
	assert.NotContains(t, ids, colleagueContact)
}

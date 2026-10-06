package controllers

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type sseClient struct {
	resp   *http.Response
	events chan string
}

// openStream abre o handler REAL de SSE e devolve um leitor de eventos.
func openStream(t *testing.T, db *gorm.DB, hub *services.SSEHub, tenantID uuid.UUID, userID int, alcance, rooms string) *sseClient {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "sse-test-secret")
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": userID, "tenantId": tenantID.String(), "alcance": alcance, "exp": 4102444800,
	}).SignedString([]byte(os.Getenv("JWT_SECRET")))
	require.NoError(t, err)

	r := gin.New()
	r.GET("/events", NewSSEController(hub, nil, db).Stream)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	url := srv.URL + "/events?token=" + tok
	if rooms != "" {
		url += "&rooms=" + rooms
	}
	resp, err := http.Get(url)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	c := &sseClient{resp: resp, events: make(chan string, 32)}
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if line := sc.Text(); strings.HasPrefix(line, "event: ") {
				c.events <- strings.TrimPrefix(line, "event: ")
			}
		}
		close(c.events)
	}()
	// espera o evento inicial "connected" (conexão já registrada no hub)
	select {
	case ev := <-c.events:
		require.Equal(t, "connected", ev)
	case <-time.After(2 * time.Second):
		t.Fatal("sem evento connected")
	}
	return c
}

func (c *sseClient) gotWithin(event string, d time.Duration) bool {
	timer := time.After(d)
	for {
		select {
		case ev, ok := <-c.events:
			if !ok {
				return false
			}
			if ev == event {
				return true
			}
		case <-timer:
			return false
		}
	}
}

func TestStream_ForeignTenantRoomQueryDoesNotLeak(t *testing.T) {
	db := testutil.NewTestDB(t)
	hub := services.NewSSEHub()
	bc := services.NewSSEBroadcast(hub)
	tenantA, tenantB := uuid.New(), uuid.New()

	c := openStream(t, db, hub, tenantA, 1, "proprio", "tenant:"+tenantB.String())

	bc.EmitToTenantRoom(tenantB.String(), "secret-b", map[string]string{"x": "da B"})
	assert.False(t, c.gotWithin("secret-b", 300*time.Millisecond), "o usuário da A não pode receber eventos da B")

	bc.EmitToTenantRoom(tenantA.String(), "own-a", map[string]string{"x": "da A"})
	assert.True(t, c.gotWithin("own-a", time.Second), "a própria sala continua entregando")
}

func TestStream_PersonalRoomIsServerDerivedAndIsolated(t *testing.T) {
	db := testutil.NewTestDB(t)
	hub := services.NewSSEHub()
	bc := services.NewSSEBroadcast(hub)
	tenant := uuid.New()

	u1 := openStream(t, db, hub, tenant, 1, "proprio", "")
	u2 := openStream(t, db, hub, tenant, 2, "proprio", "user:"+tenant.String()+":1") // tenta escutar a sala do usuário 1

	bc.EmitToRoom("/", userRoom(tenant, 1), "call.incoming", map[string]string{"x": "toque do usuario 1"})

	assert.True(t, u1.gotWithin("call.incoming", time.Second), "o dono da sala pessoal recebe")
	assert.False(t, u2.gotWithin("call.incoming", 300*time.Millisecond), "outro usuário não pode se inscrever na sala pessoal alheia pelo query")
	assert.True(t, hub.HasSubscribers(userRoom(tenant, 1)))
	assert.True(t, hub.HasSubscribers(userRoom(tenant, 2)))
	assert.False(t, hub.HasSubscribers(userRoom(tenant, 3)))
}

func TestStream_ChatRoomRespectsTicketVisibility(t *testing.T) {
	db := testutil.NewTestDB(t)
	hub := services.NewSSEHub()
	bc := services.NewSSEBroadcast(hub)
	tenant, other := uuid.New(), uuid.New()

	wa := models.Whatsapp{Name: "WA", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	contact := models.Contact{Name: "C", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(&contact).Error)
	user := models.User{Name: "Ana", Email: "ana@t.io", TenantID: tenant, Alcance: "proprio"}
	require.NoError(t, db.Create(&user).Error)

	mine := models.Ticket{Status: "open", ContactID: contact.ID, WhatsappID: wa.ID, TenantID: tenant, UserID: &user.ID}
	theirs := models.Ticket{Status: "open", ContactID: contact.ID, WhatsappID: wa.ID, TenantID: tenant}
	require.NoError(t, db.Create(&mine).Error)
	require.NoError(t, db.Create(&theirs).Error)
	foreignContact := models.Contact{Name: "F", Number: "5511999990002", TenantID: other}
	require.NoError(t, db.Create(&foreignContact).Error)
	foreignWa := models.Whatsapp{Name: "WB", TenantID: other, Status: "CONNECTED"}
	require.NoError(t, db.Create(&foreignWa).Error)
	foreign := models.Ticket{Status: "open", ContactID: foreignContact.ID, WhatsappID: foreignWa.ID, TenantID: other}
	require.NoError(t, db.Create(&foreign).Error)

	rooms := "chat:" + strconv.Itoa(mine.ID) + ",chat:" + strconv.Itoa(theirs.ID) + ",chat:" + strconv.Itoa(foreign.ID)
	c := openStream(t, db, hub, tenant, user.ID, "proprio", rooms)

	bc.EmitToRoom("/", "chat:"+strconv.Itoa(mine.ID), "mine", nil)
	bc.EmitToRoom("/", "chat:"+strconv.Itoa(theirs.ID), "theirs", nil)
	bc.EmitToRoom("/", "chat:"+strconv.Itoa(foreign.ID), "foreign", nil)

	assert.True(t, c.gotWithin("mine", time.Second), "ticket atribuído ao usuário é assinável")
	assert.False(t, c.gotWithin("theirs", 300*time.Millisecond), "ticket sem relação com o usuário não é assinável")
	assert.False(t, c.gotWithin("foreign", 300*time.Millisecond), "ticket de outra empresa nunca é assinável")
}

func TestStream_TenantScopeSeesAllTicketsOfOwnTenantOnly(t *testing.T) {
	db := testutil.NewTestDB(t)
	hub := services.NewSSEHub()
	bc := services.NewSSEBroadcast(hub)
	tenant, other := uuid.New(), uuid.New()

	wa := models.Whatsapp{Name: "WA", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	contact := models.Contact{Name: "C", Number: "5511999990003", TenantID: tenant}
	require.NoError(t, db.Create(&contact).Error)
	tk := models.Ticket{Status: "open", ContactID: contact.ID, WhatsappID: wa.ID, TenantID: tenant}
	require.NoError(t, db.Create(&tk).Error)
	fc := models.Contact{Name: "F", Number: "5511999990004", TenantID: other}
	require.NoError(t, db.Create(&fc).Error)
	fwa := models.Whatsapp{Name: "WB", TenantID: other, Status: "CONNECTED"}
	require.NoError(t, db.Create(&fwa).Error)
	ft := models.Ticket{Status: "open", ContactID: fc.ID, WhatsappID: fwa.ID, TenantID: other}
	require.NoError(t, db.Create(&ft).Error)

	c := openStream(t, db, hub, tenant, 1, "tenant", "chat:"+strconv.Itoa(tk.ID)+",chat:"+strconv.Itoa(ft.ID))
	bc.EmitToRoom("/", "chat:"+strconv.Itoa(tk.ID), "own", nil)
	bc.EmitToRoom("/", "chat:"+strconv.Itoa(ft.ID), "foreign", nil)

	assert.True(t, c.gotWithin("own", time.Second), "alcance de empresa vê qualquer ticket da própria empresa")
	assert.False(t, c.gotWithin("foreign", 300*time.Millisecond), "mas nunca de outra")
}

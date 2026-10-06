package calls

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type world struct {
	db     *gorm.DB
	tenant uuid.UUID
	other  uuid.UUID
	waA    models.Whatsapp // conexão A (vai receber a chamada)
	waB    models.Whatsapp // conexão B (sem relação)
	waX    models.Whatsapp // conexão de OUTRA empresa
	users  map[string]models.User
}

func mkUser(t *testing.T, db *gorm.DB, tenant uuid.UUID, name, alcance string, waID *int) models.User {
	t.Helper()
	u := models.User{Name: name, Email: name + "@" + tenant.String()[:8] + ".io", TenantID: tenant, Alcance: alcance, WhatsappID: waID}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func newWorld(t *testing.T) *world {
	t.Helper()
	db := testutil.NewTestDB(t)
	w := &world{db: db, tenant: uuid.New(), other: uuid.New(), users: map[string]models.User{}}
	mkWA := func(tenant uuid.UUID, name string) models.Whatsapp {
		wa := models.Whatsapp{Name: name + tenant.String(), TenantID: tenant, Status: "CONNECTED"}
		require.NoError(t, db.Create(&wa).Error)
		return wa
	}
	w.waA, w.waB, w.waX = mkWA(w.tenant, "A"), mkWA(w.tenant, "B"), mkWA(w.other, "X")

	qA := models.Queue{Name: "Fila A", Color: "#111", TenantID: w.tenant}
	qB := models.Queue{Name: "Fila B", Color: "#222", TenantID: w.tenant}
	require.NoError(t, db.Create(&qA).Error)
	require.NoError(t, db.Create(&qB).Error)
	require.NoError(t, db.Exec(`INSERT INTO whatsapp_queues (whatsapp_id, queue_id) VALUES (?, ?), (?, ?)`, w.waA.ID, qA.ID, w.waB.ID, qB.ID).Error)

	w.users["admin"] = mkUser(t, db, w.tenant, "admin", "tenant", nil)
	w.users["plataforma"] = mkUser(t, db, w.tenant, "plataforma", "plataforma", nil)
	// User.WhatsappID é só a conexão preferida para criar ticket: NÃO dá visibilidade.
	w.users["dono_da_conexao"] = mkUser(t, db, w.tenant, "dono", "proprio", &w.waA.ID)
	w.users["da_fila_A"] = mkUser(t, db, w.tenant, "filaA", "proprio", nil)
	w.users["da_fila_B"] = mkUser(t, db, w.tenant, "filaB", "proprio", nil)
	w.users["sem_nada"] = mkUser(t, db, w.tenant, "semnada", "proprio", nil)
	w.users["setor_fila_A"] = mkUser(t, db, w.tenant, "setorA", "setor", nil)
	w.users["de_outra_empresa"] = mkUser(t, db, w.other, "outra", "tenant", nil)
	require.NoError(t, db.Exec(`INSERT INTO user_queues (user_id, queue_id) VALUES (?, ?), (?, ?), (?, ?)`,
		w.users["da_fila_A"].ID, qA.ID, w.users["da_fila_B"].ID, qB.ID, w.users["setor_fila_A"].ID, qA.ID).Error)
	return w
}

func idsOf(us []models.User) []int {
	out := make([]int, 0, len(us))
	for _, u := range us {
		out = append(out, u.ID)
	}
	sort.Ints(out)
	return out
}

func TestUsersWhoSeeConnection(t *testing.T) {
	w := newWorld(t)
	got, err := UsersWhoSeeConnection(w.db, w.tenant, w.waA.ID)
	require.NoError(t, err)
	want := []int{w.users["admin"].ID, w.users["plataforma"].ID, w.users["da_fila_A"].ID, w.users["setor_fila_A"].ID}
	sort.Ints(want)
	assert.Equal(t, want, idsOf(got), "alcance de empresa + fila ligada à conexão")
	for _, name := range []string{"dono_da_conexao", "da_fila_B", "sem_nada", "de_outra_empresa"} {
		assert.NotContains(t, idsOf(got), w.users[name].ID, "%s não enxerga a conexão A", name)
	}
}

func TestUsersWhoSeeConnection_ForeignConnectionIsEmpty(t *testing.T) {
	w := newWorld(t)
	got, err := UsersWhoSeeConnection(w.db, w.tenant, w.waX.ID)
	require.NoError(t, err)
	assert.Empty(t, got, "uma conexão de outra empresa nunca devolve usuários")

	got, err = UsersWhoSeeConnection(w.db, w.tenant, 999999)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestUsersWhoSeeConnection_DoesNotReuseScopedDB(t *testing.T) {
	w := newWorld(t)
	scoped := w.db.Where(`"tenantId" = ?`, w.tenant).Where("id = ?", -1)
	got, err := UsersWhoSeeConnection(scoped, w.tenant, w.waA.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got, "um db já escopado não pode acumular condições e zerar o resultado")
}

// Paridade: para cada usuário, "enxerga a conexão A" aqui == "enxerga um ticket
// criado na conexão A" pela regra real de GetScopedDB("Tickets"). Se alguém mudar
// uma das duas regras sem a outra, este teste falha.
func TestUsersWhoSeeConnection_ParityWithTicketVisibility(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := newWorld(t)

	contact := models.Contact{Name: "C", Number: "5511999990000", TenantID: w.tenant}
	require.NoError(t, w.db.Create(&contact).Error)
	for _, wa := range []models.Whatsapp{w.waA, w.waB} {
		tk := models.Ticket{Status: "open", ContactID: contact.ID, WhatsappID: wa.ID, TenantID: w.tenant}
		require.NoError(t, w.db.Create(&tk).Error)
	}

	seesTicketOn := func(u models.User, waID int) bool {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("db", w.db)
			c.Set("tenantId", w.tenant)
			c.Set("alcance", u.Alcance)
			c.Set("userId", float64(u.ID))
			c.Next()
		})
		var n int64
		r.GET("/x", func(c *gin.Context) {
			db, _, _ := auth.GetScoped(c, "Tickets")
			db.Session(&gorm.Session{}).Model(&models.Ticket{}).Where(`"whatsappId" = ?`, waID).Count(&n)
			c.Status(http.StatusOK)
		})
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
		return n > 0
	}

	for _, wa := range []models.Whatsapp{w.waA, w.waB} {
		got, err := UsersWhoSeeConnection(w.db, w.tenant, wa.ID)
		require.NoError(t, err)
		inCalls := map[int]bool{}
		for _, u := range got {
			inCalls[u.ID] = true
		}
		for name, u := range w.users {
			if u.TenantID != w.tenant {
				continue
			}
			assert.Equal(t, seesTicketOn(u, wa.ID), inCalls[u.ID],
				"divergência de visibilidade para %q na conexão %d: tickets=%v chamadas=%v", name, wa.ID, seesTicketOn(u, wa.ID), inCalls[u.ID])
		}
	}
}

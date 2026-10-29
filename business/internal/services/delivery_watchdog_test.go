package services

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type wdBroadcast struct {
	mu     sync.Mutex
	rooms  []string
	events []string
}

func (b *wdBroadcast) EmitToRoom(_, room, event string, _ interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rooms, b.events = append(b.rooms, room), append(b.events, event)
}
func (b *wdBroadcast) EmitToTenantRoom(string, string, interface{}) {}
func (b *wdBroadcast) EmitToNamespace(string, string, interface{})  {}

type wdLock struct{ ok bool }

func (l wdLock) SetLock(string, string, time.Duration) (bool, error) { return l.ok, nil }

func TestDeliveryWatchdog(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T", Status: "active"}).Error)
	wa := models.Whatsapp{Name: "W", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	ct := models.Contact{Name: "C", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(&ct).Error)
	tk := models.Ticket{ContactID: ct.ID, WhatsappID: wa.ID, TenantID: tenant, Status: "open"}
	require.NoError(t, db.Create(&tk).Error)

	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	add := func(id string, ack int, age time.Duration, mut func(*models.Message)) {
		m := models.Message{ID: id, Body: "x", Ack: ack, FromMe: true, TicketID: tk.ID, TenantID: tenant, CreatedAt: now.Add(-age), UpdatedAt: now.Add(-age)}
		if mut != nil {
			mut(&m)
		}
		require.NoError(t, db.Create(&m).Error)
	}
	add("PRESA", 1, 3*time.Minute, nil)                                                 // sem recibo há 3 min: falha
	add("NUNCASAIU", 0, 3*time.Minute, nil)                                             // o engine nunca confirmou o envio: o caso mais grave
	add("RECENTE", 1, 30*time.Second, nil)                                              // ainda dentro do prazo
	add("ENTREGUE", 2, 10*time.Minute, nil)                                             // ack 2 não é falha
	add("LIDA", 3, 10*time.Minute, nil)                                                 // idem
	add("DOCONTATO", 1, 10*time.Minute, func(m *models.Message) { m.FromMe = false })   // recebida: nunca
	add("CHAMADA", 1, 10*time.Minute, func(m *models.Message) { m.MediaType = "call" }) // registro de chamada: não é envio
	add("ANTIGA", 1, 30*time.Hour, nil)                                                 // fora da janela de 24 h
	add("APAGADA", 1, 10*time.Minute, func(m *models.Message) { m.IsDeleted = true })   // apagada: não alarma

	b := &wdBroadcast{}
	w := NewDeliveryWatchdog(db, wdLock{ok: true}, b)
	w.now = func() time.Time { return now }

	ack := func(id string) int {
		var m models.Message
		require.NoError(t, db.Where("id = ?", id).Take(&m).Error)
		return m.Ack
	}

	t.Run("marca só a que o engine nunca enviou (ack 0 depois do prazo, enviada por nós)", func(t *testing.T) {
		assert.Equal(t, 1, w.Tick())
		assert.Equal(t, 5, ack("NUNCASAIU"))
		assert.Equal(t, 1, ack("PRESA"), "ack 1 por horas é celular offline: o WhatsApp entrega quando voltar")
		for _, id := range []string{"RECENTE", "DOCONTATO", "CHAMADA", "ANTIGA", "APAGADA"} {
			assert.Equal(t, 1, ack(id), id+" não podia mudar")
		}
		assert.Equal(t, 2, ack("ENTREGUE"))
		assert.Equal(t, 3, ack("LIDA"))
	})

	t.Run("avisa a tela do ticket certo", func(t *testing.T) {
		require.Len(t, b.rooms, 1)
		assert.Equal(t, "chat:"+strconv.Itoa(tk.ID), b.rooms[0])
		assert.Equal(t, "appMessage", b.events[0])
	})

	t.Run("segunda varredura não repete", func(t *testing.T) {
		assert.Equal(t, 0, w.Tick())
		assert.Len(t, b.rooms, 1)
	})

	t.Run("a recente (ack 0) é marcada quando o prazo dela passa", func(t *testing.T) {
		add("NOVA", 0, 30*time.Second, nil)
		assert.Equal(t, 0, w.Tick())
		w.now = func() time.Time { return now.Add(2 * time.Minute) }
		assert.Equal(t, 1, w.Tick())
		assert.Equal(t, 5, ack("NOVA"))
		assert.Equal(t, 1, ack("RECENTE"), "ack 1 nunca é marcado, por mais tempo que passe")
	})

	t.Run("outro nó dono do ciclo: não varre", func(t *testing.T) {
		add("OUTRA", 0, 10*time.Minute, nil)
		w2 := NewDeliveryWatchdog(db, wdLock{ok: false}, b)
		w2.now = func() time.Time { return now }
		assert.Equal(t, 0, w2.Tick())
		assert.Equal(t, 0, ack("OUTRA"))
	})
}

// Recibo real depois do palpite do vigia: a mensagem chegou, a falha some. O 5 do servidor (server-error) não desce.
func TestHandleMessageAck_RecibosDepoisDoVigia(t *testing.T) {
	cases := []struct {
		name      string
		atual, in int
		want      int
	}{
		{"entregue depois da falha do vigia desfaz", 5, 2, 2},
		{"lida depois da falha do vigia desfaz", 5, 3, 3},
		{"reproduzida depois da falha do vigia desfaz", 5, 4, 4},
		{"enviada (1) depois da falha desfaz: o engine só emite 1 quando o envio deu certo", 5, 1, 1},
		{"o 5 em cima de 5 continua 5", 5, 5, 5},
		{"o 5 sobe de 2 (erro do servidor)", 2, 5, 5},
		{"ack 2 não desce a lida", 3, 2, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tenant := uuid.New()
			msg := &domain.Message{ID: "m", TicketID: 1, Ack: c.atual, FromMe: true}
			mr := &ackMockMessageRepo{msg: msg}
			el := &EventListener{messages: mr, tickets: &ackMockTicketRepo{ticket: &domain.Ticket{ID: 1, TenantID: tenant}}}
			payload, _ := json.Marshal(map[string]interface{}{"messageId": "m", "ack": c.in})
			require.NoError(t, el.handleMessageAck(context.Background(), payload, tenant))
			assert.Equal(t, c.want, msg.Ack)
		})
	}
}

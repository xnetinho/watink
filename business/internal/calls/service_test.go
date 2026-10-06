package calls

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePublisher struct {
	mu   sync.Mutex
	sent []sentCmd
	err  error
}

type sentCmd struct {
	key     string
	payload map[string]interface{}
}

func (f *fakePublisher) PublishCommand(key string, payload interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	m, _ := payload.(map[string]interface{})
	f.sent = append(f.sent, sentCmd{key, m})
	return nil
}

func (f *fakePublisher) cmds(name string) []sentCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []sentCmd
	for _, c := range f.sent {
		if c.payload["type"] == name {
			out = append(out, c)
		}
	}
	return out
}

type emitted struct {
	room, event string
	payload     interface{}
}

type fakeBroadcaster struct {
	mu  sync.Mutex
	evs []emitted
}

func (f *fakeBroadcaster) EmitToRoom(_, room, event string, p interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.evs = append(f.evs, emitted{room, event, p})
}
func (f *fakeBroadcaster) EmitToTenantRoom(tenant, event string, p interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.evs = append(f.evs, emitted{"tenant:" + tenant, event, p})
}
func (f *fakeBroadcaster) EmitToNamespace(string, string, interface{}) {}
func (f *fakeBroadcaster) to(room, event string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, e := range f.evs {
		if e.room == room && e.event == event {
			n++
		}
	}
	return n
}

type fakePresence struct{ online map[string]bool }

func (f fakePresence) HasSubscribers(room string) bool { return f.online[room] }

type rig struct {
	*world
	svc  *Service
	pub  *fakePublisher
	bc   *fakeBroadcaster
	pres fakePresence
}

func newRig(t *testing.T) *rig {
	t.Helper()
	w := newWorld(t)
	r := &rig{world: w, pub: &fakePublisher{}, bc: &fakeBroadcaster{}, pres: fakePresence{online: map[string]bool{}}}
	r.svc = NewService(w.db, repository.NewGORMContactRepo(w.db), repository.NewGORMTicketRepo(w.db),
		repository.NewGORMQueueRepo(w.db), r.pub, r.bc, r.pres)
	r.svc.now = func() time.Time { return time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC) }
	return r
}

func (r *rig) online(names ...string) {
	for _, n := range names {
		r.pres.online[UserRoom(r.tenant, r.users[n].ID)] = true
	}
}

func (r *rig) grant(t *testing.T, name, action string) {
	t.Helper()
	u := r.users[name]
	var permID int
	require.NoError(t, r.db.Raw(`SELECT id FROM "Permissions" WHERE resource = 'calls' AND action = ?`, action).Scan(&permID).Error)
	if permID == 0 {
		require.NoError(t, r.db.Exec(`INSERT INTO "Permissions" (resource, action, description, "isSystem") VALUES ('calls', ?, ?, true)`, action, action).Error)
		require.NoError(t, r.db.Raw(`SELECT id FROM "Permissions" WHERE resource = 'calls' AND action = ?`, action).Scan(&permID).Error)
	}
	cargoName := "cargo-" + name
	require.NoError(t, r.db.Exec(`INSERT INTO "Cargos" (name, description, "tenantId") VALUES (?, '', ?) ON CONFLICT DO NOTHING`, cargoName, u.TenantID).Error)
	var cargoID int
	require.NoError(t, r.db.Raw(`SELECT id FROM "Cargos" WHERE name = ? AND "tenantId" = ?`, cargoName, u.TenantID).Scan(&cargoID).Error)
	require.NoError(t, r.db.Exec(`INSERT INTO cargo_permissoes ("cargoId", "permissionId") VALUES (?, ?) ON CONFLICT DO NOTHING`, cargoID, permID).Error)
	require.NoError(t, r.db.Model(&models.User{}).Where("id = ?", u.ID).Update(`"cargoId"`, cargoID).Error)
}

func incoming(callID string, wa int, peer, pn string) json.RawMessage {
	b, _ := json.Marshal(map[string]interface{}{
		"callId": callID, "sessionId": itoa(wa), "peer": peer, "callerPn": pn, "direction": "incoming", "media": "audio",
	})
	return b
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func (r *rig) log(t *testing.T, callID string) models.CallLog {
	t.Helper()
	var l models.CallLog
	require.NoError(t, r.db.Where(`"tenantId" = ? AND "callId" = ?`, r.tenant, callID).First(&l).Error)
	return l
}

func (r *rig) countLogs(t *testing.T, callID string) int64 {
	var n int64
	require.NoError(t, r.db.Model(&models.CallLog{}).Where(`"tenantId" = ? AND "callId" = ?`, r.tenant, callID).Count(&n).Error)
	return n
}

var ctx = context.Background()

// 6.1: a mesma entrega duas vezes gera um único registro.
func TestHandleIncoming_IsIdempotent(t *testing.T) {
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")
	r.online("da_fila_A")

	ev := incoming("CALL-1", r.waA.ID, "5511999990001@s.whatsapp.net", "5511999990001")
	require.NoError(t, r.svc.HandleIncoming(ctx, r.tenant, ev))
	require.NoError(t, r.svc.HandleIncoming(ctx, r.tenant, ev))

	assert.EqualValues(t, 1, r.countLogs(t, "CALL-1"), "reentrega não duplica o registro")
	assert.Len(t, r.pub.cmds("call.ready"), 1, "nem confirma duas vezes ao engine")
	assert.Equal(t, 1, r.bc.to(UserRoom(r.tenant, r.users["da_fila_A"].ID), "call.incoming"), "nem toca duas vezes")
}

// 6.2: oferta por LID com telefone conhecido cai no contato da agenda.
func TestResolveContact_LIDWithKnownPhoneUsesExistingContact(t *testing.T) {
	r := newRig(t)
	existing := models.Contact{Name: "Maria da agenda", Number: "5511988887777", TenantID: r.tenant}
	require.NoError(t, r.db.Create(&existing).Error)

	c, tk, err := r.svc.ResolveContactAndTicket(ctx, r.tenant, r.waA.ID, "123456789@lid", "5511988887777")
	require.NoError(t, err)
	assert.Equal(t, existing.ID, c.ID, "o contato da agenda é reaproveitado")
	assert.NotZero(t, tk.ID)

	var n int64
	r.db.Model(&models.Contact{}).Where(`"tenantId" = ?`, r.tenant).Count(&n)
	assert.EqualValues(t, 1, n, "nenhum segundo contato é criado")
}

func TestResolveContact_LIDWithoutPhoneCreatesContactByLID(t *testing.T) {
	r := newRig(t)
	c, _, err := r.svc.ResolveContactAndTicket(ctx, r.tenant, r.waA.ID, "123456789@lid", "")
	require.NoError(t, err)
	require.NotNil(t, c.Lid)
	assert.Equal(t, "123456789@lid", *c.Lid)

	again, _, err := r.svc.ResolveContactAndTicket(ctx, r.tenant, r.waA.ID, "123456789@lid", "")
	require.NoError(t, err)
	assert.Equal(t, c.ID, again.ID, "a segunda oferta do mesmo LID acha o mesmo contato")
}

// 6.3: só toca para quem enxerga a conexão, tem calls:receive, está online e não pausou.
func TestEligible_AllConditions(t *testing.T) {
	r := newRig(t)
	for _, n := range []string{"admin", "da_fila_A", "da_fila_B", "sem_nada", "setor_fila_A"} {
		r.grant(t, n, "receive")
	}
	r.online("admin", "da_fila_A", "da_fila_B", "sem_nada", "setor_fila_A")
	can := func(id int) bool { return r.svc.userCan(r.tenant, id, "receive") }

	users, err := r.svc.Eligible(r.tenant, r.waA.ID, can)
	require.NoError(t, err)
	got := idsOf(users)
	assert.ElementsMatch(t, []int{r.users["admin"].ID, r.users["da_fila_A"].ID, r.users["setor_fila_A"].ID}, got,
		"admin (empresa) e quem tem a fila da conexão A")
	assert.NotContains(t, got, r.users["da_fila_B"].ID, "fora do alcance da conexão")
	assert.NotContains(t, got, r.users["sem_nada"].ID, "sem fila nenhuma")
	assert.NotContains(t, got, r.users["de_outra_empresa"].ID, "de outra empresa")

	r.svc.SetPaused(r.tenant, r.users["da_fila_A"].ID, true)
	users, _ = r.svc.Eligible(r.tenant, r.waA.ID, can)
	assert.NotContains(t, idsOf(users), r.users["da_fila_A"].ID, "pausado não toca")
	assert.Contains(t, idsOf(users), r.users["setor_fila_A"].ID, "os demais seguem tocando")

	r.svc.SetPaused(r.tenant, r.users["da_fila_A"].ID, false)
	users, _ = r.svc.Eligible(r.tenant, r.waA.ID, can)
	assert.Contains(t, idsOf(users), r.users["da_fila_A"].ID, "despausar volta a tocar")
}

func TestEligible_WithoutPermissionOrOfflineIsExcluded(t *testing.T) {
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")
	r.online("da_fila_A", "setor_fila_A")
	can := func(id int) bool { return r.svc.userCan(r.tenant, id, "receive") }
	users, err := r.svc.Eligible(r.tenant, r.waA.ID, can)
	require.NoError(t, err)
	assert.NotContains(t, idsOf(users), r.users["setor_fila_A"].ID, "online mas sem calls:receive não toca")

	for k := range r.pres.online {
		delete(r.pres.online, k)
	}
	users, _ = r.svc.Eligible(r.tenant, r.waA.ID, can)
	assert.Empty(t, users, "ninguém online → ninguém elegível")
}

func TestHandleIncoming_NoEligibleDoesNotConfirmAndRegistersMissed(t *testing.T) {
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")

	require.NoError(t, r.svc.HandleIncoming(ctx, r.tenant, incoming("CALL-2", r.waA.ID, "5511999990002@s.whatsapp.net", "5511999990002")))

	assert.Empty(t, r.pub.cmds("call.ready"), "sem elegível NÃO se confirma ao engine: o celular segue tocando")
	l := r.log(t, "CALL-2")
	assert.Equal(t, StatusMissed, l.Status)
	assert.Equal(t, "no_operator", l.EndReason)
	assert.NotNil(t, l.EndedAt)
}

func TestHandleIncoming_ConfirmsAndRingsOnlyEligible(t *testing.T) {
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")
	r.grant(t, "da_fila_B", "receive")
	r.grant(t, "sem_nada", "receive")
	r.online("da_fila_A", "da_fila_B", "sem_nada")

	require.NoError(t, r.svc.HandleIncoming(ctx, r.tenant, incoming("CALL-3", r.waA.ID, "5511999990003@s.whatsapp.net", "5511999990003")))

	ready := r.pub.cmds("call.ready")
	require.Len(t, ready, 1)
	assert.Contains(t, ready[0].key, "."+itoa(r.waA.ID)+".call.ready")
	assert.Equal(t, 1, r.bc.to(UserRoom(r.tenant, r.users["da_fila_A"].ID), "call.incoming"))
	assert.Zero(t, r.bc.to(UserRoom(r.tenant, r.users["da_fila_B"].ID), "call.incoming"), "fora do alcance")
	assert.Zero(t, r.bc.to(UserRoom(r.tenant, r.users["sem_nada"].ID), "call.incoming"), "sem fila")
	assert.Equal(t, StatusRinging, r.log(t, "CALL-3").Status)
}

func TestHandleIncoming_ReadyFailureReturnsError(t *testing.T) {
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")
	r.online("da_fila_A")
	r.pub.err = assertErr("amqp fora")
	assert.Error(t, r.svc.HandleIncoming(ctx, r.tenant, incoming("CALL-4", r.waA.ID, "5511999990004@s.whatsapp.net", "5511999990004")))
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

// 6.7: perdida cria ticket pendente se não há aberto, ou usa o existente.
func TestHandleMissed_CreatesPendingTicketWhenNoneOpen(t *testing.T) {
	r := newRig(t)
	ev, _ := json.Marshal(map[string]interface{}{
		"callId": "M-1", "sessionId": itoa(r.waA.ID), "peer": "5511999990005@s.whatsapp.net", "callerPn": "5511999990005", "reason": "busy",
	})
	require.NoError(t, r.svc.HandleMissed(ctx, r.tenant, ev))

	l := r.log(t, "M-1")
	assert.Equal(t, StatusMissed, l.Status)
	assert.Equal(t, "busy", l.EndReason)
	require.NotNil(t, l.TicketID)
	var tk models.Ticket
	require.NoError(t, r.db.First(&tk, *l.TicketID).Error)
	assert.Equal(t, "pending", tk.Status)
	assert.Equal(t, r.waA.ID, tk.WhatsappID)
}

func TestHandleMissed_UsesExistingOpenTicket(t *testing.T) {
	r := newRig(t)
	contact := models.Contact{Name: "Ana", Number: "5511999990006", TenantID: r.tenant}
	require.NoError(t, r.db.Create(&contact).Error)
	open := models.Ticket{Status: "open", ContactID: contact.ID, WhatsappID: r.waA.ID, TenantID: r.tenant}
	require.NoError(t, r.db.Create(&open).Error)

	ev, _ := json.Marshal(map[string]interface{}{
		"callId": "M-2", "sessionId": itoa(r.waA.ID), "peer": "5511999990006@s.whatsapp.net", "callerPn": "5511999990006", "reason": "proxy_blocked",
	})
	require.NoError(t, r.svc.HandleMissed(ctx, r.tenant, ev))

	l := r.log(t, "M-2")
	require.NotNil(t, l.TicketID)
	assert.Equal(t, open.ID, *l.TicketID, "usa o ticket aberto, não cria outro")
	var n int64
	r.db.Model(&models.Ticket{}).Where(`"tenantId" = ? AND "contactId" = ?`, r.tenant, contact.ID).Count(&n)
	assert.EqualValues(t, 1, n)
}

func TestHandleMissed_WritesSystemMessageOnceAndEmits(t *testing.T) {
	r := newRig(t)
	ev, _ := json.Marshal(map[string]interface{}{
		"callId": "M-3", "sessionId": itoa(r.waA.ID), "peer": "5511999990007@s.whatsapp.net", "callerPn": "5511999990007", "reason": "unsupported_type",
	})
	require.NoError(t, r.svc.HandleMissed(ctx, r.tenant, ev))
	require.NoError(t, r.svc.HandleMissed(ctx, r.tenant, ev))

	l := r.log(t, "M-3")
	var msgs []models.Message
	require.NoError(t, r.db.Where(`"ticketId" = ? AND "mediaType" = 'call'`, *l.TicketID).Find(&msgs).Error)
	require.Len(t, msgs, 1, "a reentrega não cria uma segunda mensagem no ticket")
	assert.Equal(t, "Chamada de voz perdida", msgs[0].Body)
	assert.Equal(t, 1, r.bc.to("chat:"+itoa(*l.TicketID), "appMessage"))
	assert.Equal(t, 1, r.bc.to("tenant:"+r.tenant.String(), "call.ended"), "o toque some nos navegadores uma vez só")
}

func envelopeOf(tenant string, typ string, payload json.RawMessage) []byte {
	b, _ := json.Marshal(map[string]interface{}{"tenantId": tenant, "type": typ, "payload": payload})
	return b
}

func TestDispatch_RoutesByTypeAndIgnoresUnknown(t *testing.T) {
	r := newRig(t)
	r.grant(t, "da_fila_A", "receive")
	r.online("da_fila_A")

	require.NoError(t, r.svc.Dispatch(ctx, envelopeOf(r.tenant.String(), "call.incoming", incoming("DSP-1", r.waA.ID, "5511999990020@s.whatsapp.net", "5511999990020"))))
	assert.Equal(t, StatusRinging, r.log(t, "DSP-1").Status)

	require.NoError(t, r.svc.Dispatch(ctx, envelopeOf(r.tenant.String(), "call.ended", ended("DSP-1", "timeout", 0))))
	assert.Equal(t, StatusMissed, r.log(t, "DSP-1").Status)

	assert.NoError(t, r.svc.Dispatch(ctx, envelopeOf(r.tenant.String(), "call.futuro", json.RawMessage(`{}`))), "tipo desconhecido não é erro")
}

func TestDispatch_RejectsBadEnvelope(t *testing.T) {
	r := newRig(t)
	assert.Error(t, r.svc.Dispatch(ctx, []byte(`{nao é json`)))
	assert.Error(t, r.svc.Dispatch(ctx, envelopeOf("nao-e-uuid", "call.incoming", json.RawMessage(`{}`))))
	assert.Error(t, r.svc.Dispatch(ctx, envelopeOf(r.tenant.String(), "call.incoming", json.RawMessage(`{"callId":""}`))),
		"oferta sem callId/sessionId volta como erro (vai para a DLQ)")
}

// A conexão informada no evento é da empresa do envelope: um tenant não registra
// chamada na conexão de outro.
func TestDispatch_ForeignConnectionNeverRingsAnyone(t *testing.T) {
	r := newRig(t)
	r.grant(t, "admin", "receive")
	r.online("admin")
	require.NoError(t, r.svc.Dispatch(ctx, envelopeOf(r.tenant.String(), "call.incoming",
		incoming("DSP-2", r.waX.ID, "5511999990021@s.whatsapp.net", "5511999990021"))))
	assert.Empty(t, r.pub.cmds("call.ready"), "conexão de outra empresa: ninguém é elegível")
	assert.Zero(t, r.bc.to(UserRoom(r.tenant, r.users["admin"].ID), "call.incoming"))
}

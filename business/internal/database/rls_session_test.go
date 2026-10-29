package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/pkg/tenantdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type rlsFakeRedis struct{ domain.RedisService }

func (rlsFakeRedis) SetLock(string, string, time.Duration) (bool, error) { return true, nil }
func (rlsFakeRedis) DelLock(string) error                                { return nil }

type rlsFakePublisher struct{ keys []string }

func (p *rlsFakePublisher) PublishCommand(key string, _ interface{}) error {
	p.keys = append(p.keys, key)
	return nil
}

// Regressão do "Internal server error" ao clicar em Conectar: o serviço de sessão usava o handle global sem escopo de
// empresa, e o roteador RLS recusava o UPDATE. Roda contra o banco migrado de verdade (papel sem BYPASSRLS).
func TestStartWhatsAppSession_ComRLSReal(t *testing.T) {
	admin, app, _, _ := migratedDB(t)
	tenant := uuid.New()
	_, err := admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'A','active',now(),now())`, tenant)
	require.NoError(t, err)
	require.NoError(t, seedRow(admin, "Whatsapps", tenant))
	var id int
	require.NoError(t, admin.QueryRow(`SELECT id FROM "Whatsapps" WHERE "tenantId" = $1`, tenant).Scan(&id))

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: routerOver(t, app)}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)

	pub := &rlsFakePublisher{}
	wss := services.NewWhatsAppSessionService(gdb, pub, rlsFakeRedis{}, domain.BroadcastOrNop(nil))
	wa := models.Whatsapp{ID: id, TenantID: tenant, Status: "DISCONNECTED", EngineType: "whatsmeow"}

	require.NoError(t, wss.StartWhatsAppSession(wa, false, "", true))
	assert.Len(t, pub.keys, 1, "o comando session.start precisa chegar ao broker")

	var status string
	require.NoError(t, admin.QueryRow(`SELECT status FROM "Whatsapps" WHERE id = $1`, id).Scan(&status))
	assert.Equal(t, "OPENING", status)

}

// Regressão: o voto de um contato chegava do barramento sem escopo de empresa e o RLS recusava o INSERT em PollResults
// ("SEM ESCOPO"), então nenhum voto aparecia. Banco migrado de verdade, papel sem BYPASSRLS.
func TestHandlePollVote_ComRLSReal(t *testing.T) {
	admin, app, _, _ := migratedDB(t)
	tenant := uuid.New()
	_, err := admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'A','active',now(),now())`, tenant)
	require.NoError(t, err)
	require.NoError(t, seedRow(admin, "Whatsapps", tenant))
	var waID int
	require.NoError(t, admin.QueryRow(`SELECT id FROM "Whatsapps" WHERE "tenantId" = $1`, tenant).Scan(&waID))
	require.NoError(t, seedRow(admin, "Contacts", tenant))
	var contactID int
	require.NoError(t, admin.QueryRow(`SELECT id FROM "Contacts" WHERE "tenantId" = $1`, tenant).Scan(&contactID))
	require.NoError(t, seedRow(admin, "Tickets", tenant))
	var ticketID int
	require.NoError(t, admin.QueryRow(`SELECT id FROM "Tickets" WHERE "tenantId" = $1`, tenant).Scan(&ticketID))

	rich := `{"rich":{"question":"Q?","options":["Sim","Não"],"selectable":1}}`
	_, err = admin.Exec(`INSERT INTO "Messages" (id, body, "mediaType", "ticketId", "tenantId", "dataJson", "createdAt", "updatedAt")
		VALUES ('POLL1','', 'poll', $1, $2, $3::jsonb, now(), now())`, ticketID, tenant, rich)
	require.NoError(t, err)

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: routerOver(t, app)}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	el := services.NewEventListener(nil, repository.NewGORMMessageRepo(gdb), nil, nil, nil, nil, gdb, nil, nil, nil, nil)

	h := sha256.Sum256([]byte("Não"))
	payload, _ := json.Marshal(map[string]string{"pollMessageId": "POLL1", "voterJid": "5511999@s.whatsapp.net", "optionSelected": hex.EncodeToString(h[:])})
	require.NoError(t, el.HandlePollVoteEvent(context.Background(), payload, tenant))

	var data string
	require.NoError(t, admin.QueryRow(`SELECT "dataJson"::text FROM "Messages" WHERE id = 'POLL1'`).Scan(&data))
	var d struct {
		Votes map[string][]string `json:"votes"`
	}
	require.NoError(t, json.Unmarshal([]byte(data), &d))
	assert.Equal(t, []string{"Não"}, d.Votes["5511999@s.whatsapp.net"], "o voto precisa ser decodificado do hash e gravado na enquete")
	_ = contactID
	_ = waID
}

type capturePublisher struct{ payloads []map[string]interface{} }

func (c *capturePublisher) PublishCommand(_ string, payload interface{}) error {
	b, _ := json.Marshal(payload)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	c.payloads = append(c.payloads, m)
	return nil
}

// Regressão: ao reiniciar, o engine anuncia engine.ready e o business restaura as sessões com session.start. Esse
// comando NÃO levava a política de presença, então toda conexão voltava como "never" depois de um restart do engine.
func TestEngineReady_RestauraSessaoComAPoliticaDePresenca(t *testing.T) {
	admin, app, system, _ := migratedDB(t)
	tenant := uuid.New()
	_, err := admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'A','active',now(),now())`, tenant)
	require.NoError(t, err)
	require.NoError(t, seedRow(admin, "Whatsapps", tenant))
	_, err = admin.Exec(`UPDATE "Whatsapps" SET status='CONNECTED', "presencePolicy"='always', "presenceIdleMinutes"=17 WHERE "tenantId"=$1`, tenant)
	require.NoError(t, err)

	router, err := tenantdb.NewRouter(app, system) // restaurar sessões descobre as empresas pelo pool de sistema
	require.NoError(t, err)
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: router}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	pub := &capturePublisher{}
	el := services.NewEventListener(repository.NewGORMChannelSessionRepo(gdb), repository.NewGORMMessageRepo(gdb), nil, nil, nil, nil, gdb, nil, nil, pub, nil)

	require.NoError(t, el.HandleEngineReady(context.Background()))

	var started []map[string]interface{}
	for _, p := range pub.payloads {
		if p["type"] == "session.start" {
			started = append(started, p["payload"].(map[string]interface{}))
		}
	}
	require.NotEmpty(t, started, "a sessão CONNECTED precisa ser restaurada")
	assert.Equal(t, "always", started[0]["presencePolicy"])
	assert.EqualValues(t, 17, started[0]["presenceIdleMinutes"])
}

// Identificação do ticket (canal + conexão) com RLS real: o enriquecimento de evento roda no escopo da empresa e NUNCA
// devolve o nome de uma conexão de outra empresa, mesmo que o id do ticket aponte para ela.
func TestTicketConnectionIdentity_ComRLSReal(t *testing.T) {
	admin, app, _, _ := migratedDB(t)
	a, b := uuid.New(), uuid.New()
	for _, tn := range []uuid.UUID{a, b} {
		_, err := admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'T','active',now(),now())`, tn)
		require.NoError(t, err)
		require.NoError(t, seedRow(admin, "Whatsapps", tn))
	}
	_, err := admin.Exec(`UPDATE "Whatsapps" SET name='Conexao A' WHERE "tenantId"=$1`, a)
	require.NoError(t, err)
	_, err = admin.Exec(`UPDATE "Whatsapps" SET name='Conexao B (segredo)' WHERE "tenantId"=$1`, b)
	require.NoError(t, err)
	var waA, waB int
	require.NoError(t, admin.QueryRow(`SELECT id FROM "Whatsapps" WHERE "tenantId"=$1`, a).Scan(&waA))
	require.NoError(t, admin.QueryRow(`SELECT id FROM "Whatsapps" WHERE "tenantId"=$1`, b).Scan(&waB))

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: routerOver(t, app)}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)

	err = tenantdb.WithTenant(context.Background(), gdb, a, func(ctx context.Context) error {
		db := gdb.WithContext(ctx)
		c, ok := services.ConnectionOf(db, 1, waA, nil)
		assert.True(t, ok)
		assert.Equal(t, "Conexao A", c.Name)
		assert.Equal(t, "whatsapp", c.Kind)

		_, ok = services.ConnectionOf(db, 2, waB, nil) // ticket da empresa A apontando para conexão da empresa B
		assert.False(t, ok, "o nome da conexão de outra empresa nunca pode vazar")
		return nil
	})
	require.NoError(t, err)
}

// Regressão: o enriquecimento do ticket em EVENTO (mensagem recebida) rodava sem escopo de empresa e o RLS recusava a
// consulta ("SEM ESCOPO"), então o ticket que nascia por mensagem aparecia sem identificação. Caminho do evento, banco real.
func TestWithConnectionIdentity_EventoComRLSReal(t *testing.T) {
	admin, app, _, _ := migratedDB(t)
	tenant := uuid.New()
	_, err := admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'T','active',now(),now())`, tenant)
	require.NoError(t, err)
	require.NoError(t, seedRow(admin, "ChannelConnections", tenant))
	_, err = admin.Exec(`UPDATE "ChannelConnections" SET kind='telegram', name='Jarvis' WHERE "tenantId"=$1`, tenant)
	require.NoError(t, err)
	var connID int
	require.NoError(t, admin.QueryRow(`SELECT id FROM "ChannelConnections" WHERE "tenantId"=$1`, tenant).Scan(&connID))

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: routerOver(t, app)}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)

	ticket := &domain.Ticket{ID: 9, TenantID: tenant, ChannelConnectionID: &connID}
	err = tenantdb.WithTenant(context.Background(), gdb, tenant, func(ctx context.Context) error {
		out, ok := services.WithConnectionIdentity(ctx, gdb, ticket).(map[string]interface{})
		require.True(t, ok, "com escopo, devolve o ticket enriquecido")
		assert.Equal(t, "telegram", out["channel"])
		conn, _ := out["connection"].(models.TicketConnectionInfo)
		assert.Equal(t, "Jarvis", conn.Name)
		return nil
	})
	require.NoError(t, err)

	// Sem escopo a consulta é recusada: o ticket sai sem "connection" (a tela mostra só o canal) e a mensagem nunca falha.
	out, ok := services.WithConnectionIdentity(context.Background(), gdb, ticket).(map[string]interface{})
	require.True(t, ok)
	_, has := out["connection"]
	assert.False(t, has, "sem escopo de empresa, nada de identificação (e nada vaza)")
}

// Importar contatos e receber o resultado da gravação no celular, com o papel SEM BYPASSRLS: cada contato nasce ligado à
// conexão que o trouxe, o mesmo telefone em outra conexão é outro contato, e o resultado só altera o contato da conexão.
func TestContatosPorConexao_ImportEResultadoDaGravacao_ComRLSReal(t *testing.T) {
	admin, app, _, _ := migratedDB(t)
	tenant := uuid.New()
	_, err := admin.Exec(`INSERT INTO "Tenants" (id, name, status, "createdAt", "updatedAt") VALUES ($1,'A','active',now(),now())`, tenant)
	require.NoError(t, err)
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: routerOver(t, app)}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	repo := repository.NewGORMContactRepo(gdb)

	err = tenantdb.WithTenant(context.Background(), gdb, tenant, func(ctx context.Context) error {
		imp := func(session string) json.RawMessage {
			b, _ := json.Marshal(map[string]interface{}{"sessionId": session, "contacts": []map[string]string{{"number": "5583999990001", "name": "Maria"}}})
			return b
		}
		require.NoError(t, services.HandleContactImportEvent(ctx, repo, imp("1"), tenant))
		require.NoError(t, services.HandleContactImportEvent(ctx, repo, imp("2"), tenant))
		require.NoError(t, services.HandleContactImportEvent(ctx, repo, imp("1"), tenant), "reimportar não duplica")
		assert.Error(t, services.HandleContactImportEvent(ctx, repo, imp(""), tenant), "importação sem conexão é recusada, nunca grava contato órfão")

		a, err := repo.FindByNumber(ctx, tenant, domain.WhatsappConnection(1), "5583999990001", false)
		require.NoError(t, err)
		b, err := repo.FindByNumber(ctx, tenant, domain.WhatsappConnection(2), "5583999990001", false)
		require.NoError(t, err)
		require.NotNil(t, a)
		require.NotNil(t, b)
		assert.NotEqual(t, a.ID, b.ID, "mesmo telefone em duas conexões = dois contatos")

		saved, _ := json.Marshal(map[string]interface{}{"sessionId": "2", "number": "5583999990001", "ok": false, "error": "5583999990001 não tem WhatsApp"})
		require.NoError(t, services.HandleContactSavedEvent(ctx, repo, saved, tenant))
		a, _ = repo.FindByNumber(ctx, tenant, domain.WhatsappConnection(1), "5583999990001", false)
		b, _ = repo.FindByNumber(ctx, tenant, domain.WhatsappConnection(2), "5583999990001", false)
		assert.Equal(t, "failed", b.PhoneSync)
		assert.Contains(t, b.PhoneSyncError, "não tem WhatsApp")
		assert.Empty(t, a.PhoneSync, "o resultado vale só para a conexão que o gerou")
		return nil
	})
	require.NoError(t, err)

	var n int
	require.NoError(t, admin.QueryRow(`SELECT count(*) FROM "Contacts" WHERE "tenantId" = $1`, tenant).Scan(&n))
	assert.Equal(t, 2, n)
}

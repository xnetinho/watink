package database

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/alltomatos/watinkdev/business/internal/knowledge"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"gorm.io/gorm"
)

var DB *gorm.DB

// MigrateVersioned applies the greenfield schema baseline exactly once per database.
// The advisory lock is acquired by RunMigrations before this function; the version row makes
// reruns observable and prevents silently treating a partial future migration as complete.
func MigrateVersioned(ctx context.Context, db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS "SchemaMigrations" ("version" bigint PRIMARY KEY, "appliedAt" timestamptz NOT NULL DEFAULT NOW())`).Error; err != nil {
		return err
	}
	var applied bool
	if err := db.Raw(`SELECT EXISTS (SELECT 1 FROM "SchemaMigrations" WHERE "version" = 1)`).Scan(&applied).Error; err != nil {
		return err
	}
	if applied {
		return nil
	}
	if err := migrateBaseline(db); err != nil {
		return err
	}
	return db.Exec(`INSERT INTO "SchemaMigrations" ("version") VALUES (1) ON CONFLICT ("version") DO NOTHING`).Error
}

func migrateBaseline(db *gorm.DB) error {
	previous := DB
	DB = db
	defer func() { DB = previous }()
	Migrate()
	return nil
}

func Migrate() {
	if err := dropLegacyRBAC(); err != nil {
		log.Printf("Warning: failed to drop legacy RBAC schema: %v", err)
	}

	err := DB.AutoMigrate(
		&models.Plan{},
		&models.Tenant{},
		&models.TenantSubscription{},
		&models.User{},
		&models.Setting{},
		&models.Contact{},
		&models.ContactIdentity{},
		&models.Client{},
		&models.ClientAddress{},
		&models.Whatsapp{},
		&models.ChannelConnection{},
		&models.WhatsAppConnectionDetails{},
		&models.Queue{},
		&models.Ticket{},
		&models.Message{},
		&models.Setor{},
		&models.Cargo{},
		&models.Permission{},
		&models.CargoPermissao{},
		&models.UserSetor{},
		&models.SetorFila{},
		&models.Flow{},
		&models.FlowRun{},
		&models.FlowRunLog{},
		&models.QuickAnswer{},
		&models.KnowledgeBase{},
		&models.KnowledgeBaseSource{},
		&models.Pipeline{},
		&models.PipelineStage{},
		&models.Deal{},
		&models.TagGroup{},
		&models.Tag{},
		&models.EntityTag{},
		&models.TicketLog{},
		&models.ConversationEmbedding{},
		&models.PollResult{},
		&models.Proxy{},
		&models.ProxyGroup{},
		&models.ConnectionGroup{},
		&models.PluginInstallation{},
		&models.Protocol{},
		&models.ProtocolLog{},
		&models.ProtocolAttachment{},
		&models.Activity{},
		&models.ActivityAssignee{},
		&models.ActivityChecklistItem{},
		&models.ActivityMaterial{},
		&models.ActivityOccurrence{},
		&models.OutboxEvent{},
		&models.Assistant{},
		&models.AssistantRouterOption{},
		&models.AssistantGroup{},
		&models.AiGateway{},
		&models.AssistantProactiveLog{},
		&models.GroupWatchTag{},
		&models.GroupWatchMatch{},
		&models.GroupCache{},
		&models.GroupCampaign{},
		&models.GroupCampaignVariant{},
		&models.GroupCampaignTarget{},
		&models.GroupCampaignRun{},
		&models.GroupCampaignSend{},
		&models.GroupCampaignReply{},
		&models.InstancePolicy{},
		&models.Product{},
		&models.ProductSKU{},
		&models.ProductComposition{},
		&models.PriceTable{},
		&models.SKUPrice{},
		&models.Warehouse{},
		&models.WarehouseBalance{},
		&models.InventoryMovement{},
		&models.CallLog{},
		&models.CallRecordingAccess{},
	)

	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := dropIzapiaSchema(DB); err != nil {
		log.Fatalf("Failed to drop retired izapia schema: %v", err)
	}

	if err := addCustomIndexes(); err != nil {
		log.Printf("Warning: failed to create custom indexes: %v", err)
	}

	// Tickets de canais não-WhatsApp gravam whatsappId = 0 (sentinel: "sem WhatsApp").
	// A FK legada fk_Whatsapps_tickets rejeitava esse valor e derrubava todo o
	// inbound de canal (violação SQLSTATE 23503). WhatsApp continua validado na
	// aplicação (resolveConnection).
	if err := DB.Exec(`ALTER TABLE "Tickets" DROP CONSTRAINT IF EXISTS "fk_Whatsapps_tickets"`).Error; err != nil {
		log.Printf("Warning: failed to drop legacy fk_Whatsapps_tickets: %v", err)
	}

	addClientAddressGeography()

	if err := BackfillContactIdentities(DB); err != nil {
		log.Printf("Warning: failed to backfill contact identities: %v", err)
	}

	if err := BackfillTicketConversationKeys(DB); err != nil {
		log.Printf("Warning: failed to backfill ticket conversation keys: %v", err)
	}

	// KBChunk (halfvec/HNSW) is not GORM-modelable, same reasoning as
	// addCustomIndexes()/PostGIS above — raw idempotent DDL, best-effort.
	knowledge.EnsureSchema(DB)

	fmt.Println("Database migration completed")
	Seed()
}

// dropIzapiaSchema normalizes any legacy engineType marker and removes the tables/columns that
// belonged to the retired external izapia engine. Active izapia connections were confirmed absent
// from production/homologation before this idempotent startup cleanup was approved.
// AutoMigrate does not drop removed model fields, hence the explicit DDL.
func dropIzapiaSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE "Whatsapps" SET "engineType" = 'whatsmeow' WHERE "engineType" = 'izapia'`).Error; err != nil {
			return err
		}
		for _, ddl := range []string{
			`DROP TABLE IF EXISTS "IzapiaConfigs"`,
			`ALTER TABLE "Whatsapps" DROP COLUMN IF EXISTS "izapiaSessionId"`,
			`ALTER TABLE "Whatsapps" DROP COLUMN IF EXISTS "izapiaWebhookSecretEnc"`,
		} {
			if err := tx.Exec(ddl).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// addClientAddressGeography adds the PostGIS spatial column backing
// ClientAddress (ADR 0023) via raw SQL — GORM does not model the `geography`
// type natively. Best-effort: the PostGIS extension may be absent in some dev
// environments, so a failure is logged and never blocks boot (log.Fatalf is
// deliberately not used here).
func addClientAddressGeography() {
	if err := DB.Exec(`ALTER TABLE "ClientAddresses" ADD COLUMN IF NOT EXISTS geog geography(Point,4326)`).Error; err != nil {
		log.Printf("Warning: failed to add ClientAddresses.geog (PostGIS extension may be missing): %v", err)
	}
}

// BackfillContactIdentities cria registros em ContactIdentities para todos os Contacts existentes
// com number ou lid válidos, atribuindo channel='whatsapp'. É 100% idempotente (ON CONFLICT DO NOTHING).
// Task 2.1 (ADR 0033).
func BackfillContactIdentities(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Backfill por number
		qNumber := `
			INSERT INTO "ContactIdentities" ("tenantId", "contactId", "channel", "externalId", "displayName", "avatarUrl", "meta", "createdAt", "updatedAt")
			SELECT c."tenantId", c.id, 'whatsapp', c.number, c.name, COALESCE(c."profilePicUrl", ''), '{}'::jsonb, NOW(), NOW()
			FROM "Contacts" c
			WHERE c.number IS NOT NULL AND c.number != ''
			ON CONFLICT ("tenantId", "channel", "externalId") DO NOTHING;
		`
		if err := tx.Exec(qNumber).Error; err != nil {
			return err
		}

		// Backfill por lid
		qLid := `
			INSERT INTO "ContactIdentities" ("tenantId", "contactId", "channel", "externalId", "displayName", "avatarUrl", "meta", "createdAt", "updatedAt")
			SELECT c."tenantId", c.id, 'whatsapp', c.lid, c.name, COALESCE(c."profilePicUrl", ''), '{}'::jsonb, NOW(), NOW()
			FROM "Contacts" c
			WHERE c.lid IS NOT NULL AND c.lid != ''
			ON CONFLICT ("tenantId", "channel", "externalId") DO NOTHING;
		`
		return tx.Exec(qLid).Error
	})
}

// BackfillTicketConversationKeys preenche conversationKey com contactId::text para tickets legados
// que ainda não possuem conversationKey definida (Task 2.3 / ADR 0033). É 100% idempotente.
func BackfillTicketConversationKeys(db *gorm.DB) error {
	q := `
		UPDATE "Tickets"
		SET "conversationKey" = "contactId"::text
		WHERE "conversationKey" IS NULL OR "conversationKey" = '';
	`
	return db.Exec(q).Error
}

// Seed recria o catálogo de Permissions em granularidade recurso:ação
// (ADR 0022) — não mais resource:view de menu. Cargos-padrão e o primeiro
// Administrador são criados por SetupService.InitializeTenant, não aqui.
func Seed() {
	permissions := []models.Permission{
		// users
		{Resource: "users", Action: "read", Description: "Visualizar Usuários"},
		{Resource: "users", Action: "create", Description: "Criar Usuários"},
		{Resource: "users", Action: "update", Description: "Editar Usuários"},
		{Resource: "users", Action: "delete", Description: "Excluir Usuários"},
		{Resource: "users", Action: "manage", Description: "Gerenciar Cargo/Setor de outros usuários"},
		// setores
		{Resource: "setores", Action: "read", Description: "Visualizar Setores"},
		{Resource: "setores", Action: "create", Description: "Criar Setores"},
		{Resource: "setores", Action: "update", Description: "Editar Setores"},
		{Resource: "setores", Action: "delete", Description: "Excluir Setores"},
		{Resource: "setores", Action: "manage", Description: "Gerenciar membros/gestores de Setores"},
		// cargos
		{Resource: "cargos", Action: "read", Description: "Visualizar Cargos"},
		{Resource: "cargos", Action: "create", Description: "Criar Cargos"},
		{Resource: "cargos", Action: "update", Description: "Editar Cargos"},
		{Resource: "cargos", Action: "delete", Description: "Excluir Cargos"},
		{Resource: "cargos", Action: "manage", Description: "Gerenciar permissões de Cargos"},
		// tickets
		{Resource: "tickets", Action: "read", Description: "Visualizar Tickets"},
		{Resource: "tickets", Action: "create", Description: "Criar Tickets"},
		{Resource: "tickets", Action: "update", Description: "Editar Tickets"},
		{Resource: "tickets", Action: "delete", Description: "Excluir Tickets"},
		{Resource: "tickets", Action: "reassign", Description: "Transferir/reatribuir Tickets"},
		{Resource: "tickets", Action: "close", Description: "Encerrar Tickets"},
		{Resource: "tickets", Action: "export", Description: "Exportar Tickets"},
		// contacts
		{Resource: "contacts", Action: "read", Description: "Visualizar Contatos"},
		{Resource: "contacts", Action: "create", Description: "Criar Contatos"},
		{Resource: "contacts", Action: "update", Description: "Editar Contatos"},
		{Resource: "contacts", Action: "delete", Description: "Excluir Contatos"},
		// clients (CRM)
		{Resource: "clients", Action: "read", Description: "Visualizar Clientes"},
		{Resource: "clients", Action: "create", Description: "Criar Clientes"},
		{Resource: "clients", Action: "update", Description: "Editar Clientes"},
		{Resource: "clients", Action: "delete", Description: "Excluir Clientes"},
		{Resource: "clients", Action: "manage", Description: "Gerenciar vínculos de Contato e Endereços de Clientes"},
		// activities (Ordens de Serviço)
		{Resource: "activities", Action: "read", Description: "Visualizar Atividades"},
		{Resource: "activities", Action: "create", Description: "Criar Atividades"},
		{Resource: "activities", Action: "update", Description: "Editar/executar Atividades"},
		{Resource: "activities", Action: "delete", Description: "Excluir Atividades"},
		{Resource: "activities", Action: "manage", Description: "Gerenciar SLA e atribuição de Atividades"},
		// inventory (WMS — catálogo, movimentações; Modo Avançado usa o mesmo
		// recurso via o plugin inventory-advanced, gate de licença é dele)
		{Resource: "inventory", Action: "read", Description: "Visualizar Estoque"},
		{Resource: "inventory", Action: "create", Description: "Criar Produtos"},
		{Resource: "inventory", Action: "update", Description: "Editar Produtos"},
		{Resource: "inventory", Action: "delete", Description: "Excluir Produtos"},
		{Resource: "inventory", Action: "manage", Description: "Registrar movimentações de estoque"},
		// connections (WhatsApp/Conexões)
		{Resource: "connections", Action: "read", Description: "Visualizar Conexões"},
		{Resource: "connections", Action: "create", Description: "Criar Conexões"},
		{Resource: "connections", Action: "update", Description: "Editar Conexões"},
		{Resource: "connections", Action: "delete", Description: "Excluir Conexões"},
		// pipelines
		{Resource: "pipelines", Action: "read", Description: "Visualizar Pipelines"},
		{Resource: "pipelines", Action: "create", Description: "Criar Pipelines"},
		{Resource: "pipelines", Action: "update", Description: "Editar Pipelines"},
		{Resource: "pipelines", Action: "delete", Description: "Excluir Pipelines"},
		// knowledgeBases (RAG)
		{Resource: "knowledgeBases", Action: "read", Description: "Visualizar Bases de Conhecimento e testar recuperação"},
		{Resource: "knowledgeBases", Action: "manage", Description: "Criar/editar/excluir Bases de Conhecimento e suas fontes"},
		// flows
		{Resource: "flows", Action: "read", Description: "Visualizar/gerenciar Flows (Automação)"},
		{Resource: "flows", Action: "create", Description: "Criar Flows"},
		{Resource: "flows", Action: "update", Description: "Editar Flows"},
		{Resource: "flows", Action: "delete", Description: "Excluir Flows"},
		// settings (inclui faturamento/billing por ora)
		{Resource: "settings", Action: "read", Description: "Visualizar Configurações"},
		{Resource: "settings", Action: "update", Description: "Editar Configurações"},
		// reports (escopo distinto)
		{Resource: "reports", Action: "view-sector", Description: "Visualizar Relatórios do Setor"},
		{Resource: "reports", Action: "view-tenant", Description: "Visualizar Relatórios do Tenant"},
		// queues
		{Resource: "queues", Action: "read", Description: "Visualizar Filas"},
		{Resource: "queues", Action: "create", Description: "Criar Filas"},
		{Resource: "queues", Action: "update", Description: "Editar Filas"},
		{Resource: "queues", Action: "delete", Description: "Excluir Filas"},
		// quickAnswers (respostas rápidas): ler/enviar é do atendimento; gerir o catálogo é separado
		{Resource: "quickAnswers", Action: "read", Description: "Visualizar Respostas Rápidas"},
		{Resource: "quickAnswers", Action: "create", Description: "Criar Respostas Rápidas"},
		{Resource: "quickAnswers", Action: "update", Description: "Editar Respostas Rápidas"},
		{Resource: "quickAnswers", Action: "delete", Description: "Excluir Respostas Rápidas"},
		// tags
		{Resource: "tags", Action: "read", Description: "Visualizar Tags"},
		{Resource: "tags", Action: "create", Description: "Criar Tags"},
		{Resource: "tags", Action: "update", Description: "Editar Tags"},
		{Resource: "tags", Action: "delete", Description: "Excluir Tags"},
		// assistants / aiGateways (plugin "assistant")
		{Resource: "assistants", Action: "read", Description: "Visualizar Assistentes de IA"},
		{Resource: "assistants", Action: "manage", Description: "Criar/editar/excluir/testar Assistentes de IA e seus gateways"},
		// helpdesk (plugin "helpdesk")
		{Resource: "protocols", Action: "read", Description: "Visualizar Protocolos do Helpdesk"},
		{Resource: "protocols", Action: "manage", Description: "Criar/editar Protocolos e anexos do Helpdesk"},
		// inventário avançado (plugin "inventory-advanced"): leitura reaproveita inventory:read; escrita é inventory:manage
		// whatsappGroups (plugin "groups" — gestão de grupos/comunidades WhatsApp)
		{Resource: "whatsappGroups", Action: "read", Description: "Visualizar grupos e comunidades do WhatsApp"},
		{Resource: "whatsappGroups", Action: "manage", Description: "Criar/configurar grupos e comunidades, vincular/desvincular subgrupos"},
		{Resource: "whatsappGroups", Action: "admin", Description: "Gerenciar participantes (adicionar/remover/promover/rebaixar), aprovar solicitações de entrada, sair de grupos"},
		// calls (chamadas de voz do WhatsApp). Nenhuma é anexada a cargo do seed:
		// só o alcance de empresa (Administrador/Gerente Geral) passa sozinho.
		{Resource: "calls", Action: "receive", Description: "Atender e recusar chamadas de voz do WhatsApp"},
		{Resource: "calls", Action: "place", Description: "Efetuar chamadas de voz pelo WhatsApp"},
		{Resource: "calls", Action: "read", Description: "Consultar o histórico de chamadas e ouvir gravações"},
		{Resource: "calls", Action: "delete", Description: "Excluir gravações de chamadas"},
		{Resource: "calls", Action: "manage", Description: "Configurar a gravação de chamadas"},
		// swagger
		{Resource: "swagger", Action: "view", Description: "Visualizar documentação Swagger"},
	}

	for _, p := range permissions {
		DB.FirstOrCreate(&p, models.Permission{Resource: p.Resource, Action: p.Action})
	}

	fmt.Println("Database seeding completed")

	if err := backfillActivitiesReadForAtendente(); err != nil {
		log.Printf("Warning: failed to backfill activities:read for Atendente cargos: %v", err)
	}
}

// backfillActivitiesReadForAtendente concede activities:read ao Cargo
// "Atendente" de todo tenant já existente. SetupService.InitializeTenant só
// anexa a permissão a tenants criados DEPOIS que ela entrou em
// atendentePermNames — sem este backfill, o técnico de um tenant já
// provisionado nunca vê o item de menu nem consegue acessar /my-activities
// (403 no dia 1). Idempotente via ON CONFLICT DO NOTHING sobre a PK composta
// (cargoId, permissionId) de cargo_permissoes — rodar de novo não duplica.
func backfillActivitiesReadForAtendente() error {
	var perm models.Permission
	if err := DB.Where(`resource = ? AND action = ?`, "activities", "read").First(&perm).Error; err != nil {
		return fmt.Errorf("activities:read permission not found: %w", err)
	}

	return DB.Exec(`
		INSERT INTO cargo_permissoes ("cargoId", "permissionId")
		SELECT c.id, ?
		FROM "Cargos" c
		WHERE c.name = 'Atendente'
		ON CONFLICT DO NOTHING
	`, perm.ID).Error
}

// dropLegacyRBAC remove o schema RBAC legado (Group/Role/RolePermission +
// tabelas de junção mortas) num reset destrutivo autorizado em dev — sem
// migração de dado (ADR 0022). Cada statement roda isolado; erros de
// "does not exist" são esperados e ignorados, outros erros são logados mas
// não interrompem a migração (fail-forward para não travar boot em dev).
func dropLegacyRBAC() error {
	statements := []string{
		`DROP TABLE IF EXISTS group_permissions CASCADE`,
		`DROP TABLE IF EXISTS group_roles CASCADE`,
		`DROP TABLE IF EXISTS user_permissions CASCADE`,
		`DROP TABLE IF EXISTS user_roles CASCADE`,
		`DROP TABLE IF EXISTS role_permissions CASCADE`,
		`ALTER TABLE "Users" DROP COLUMN IF EXISTS profile`,
		`ALTER TABLE "Users" DROP COLUMN IF EXISTS "groupId"`,
		`DROP TABLE IF EXISTS "Groups" CASCADE`,
		`DROP TABLE IF EXISTS "Roles" CASCADE`,
		`DROP TABLE IF EXISTS "RolePermissions" CASCADE`,
		// Catálogo legado de granularidade MENU (resource:view) — Seed() só usa
		// FirstOrCreate (nunca remove), então sem isso essas entradas ficam
		// poluindo o catálogo novo recurso:ação para sempre. DELETE seletivo (não
		// TRUNCATE da tabela toda) para não apagar cargo_permissoes já associadas
		// ao catálogo novo em bootups subsequentes.
		//
		// NOTA: ('flows','read') do catálogo antigo NÃO entra aqui — colide
		// (mesmo resource+action) com a permissão nova 'flows:read' (ação real,
		// não menu) criada pelo Seed(). Incluí-la aqui apagava a permissão nova
		// a cada boot (DELETE por resource+action, sem distinguir a intenção),
		// e a cascata removia o vínculo cargo_permissoes do Administrador —
		// bug real observado: Administrador ficava com 3/4 permissions de
		// flows após um segundo restart do servidor.
		`DELETE FROM "Permissions" WHERE (resource, action) IN (
			('admin','view'), ('chats','view'), ('groups','view'), ('pipelines','view'),
			('queues','view'), ('settings','view'), ('view','swagger')
		)`,
	}

	for _, stmt := range statements {
		if err := DB.Exec(stmt).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
				continue
			}
			log.Printf("dropLegacyRBAC: statement failed (%s): %v", stmt, err)
		}
	}
	return nil
}

func addCustomIndexes() error {
	// Migração multi-tenant: Queue.name/color eram unique GLOBAL
	// (uni_Queues_name/uni_Queues_color) — o unique global quebrava o
	// provisionamento do 2º tenant (dois tenants não podiam ter uma fila
	// "Atendimento Inicial" nem a mesma cor). Agora name é único por-tenant
	// (idx_queues_tenant_name, criado pelo AutoMigrate) e color não é único.
	// Dropa as constraints antigas (best-effort; ausência é esperada em banco novo).
	for _, ddl := range []string{
		`ALTER TABLE "Queues" DROP CONSTRAINT IF EXISTS uni_Queues_name`,
		`ALTER TABLE "Queues" DROP CONSTRAINT IF EXISTS uni_Queues_color`,
	} {
		if err := DB.Exec(ddl).Error; err != nil {
			log.Printf("addCustomIndexes (drop legacy Queue uniques): %q: %v", ddl, err)
		}
	}

	// Migração multi-tenant: Contacts.number/lid eram unique GLOBAL
	// (uni_Contacts_number/uni_Contacts_lid), então dois tenants não podiam ter o
	// mesmo contato e a duplicata virava erro 500. Passam a ser únicos por tenant.
	// Contatos de grupo e de conversa 1:1 compartilham a coluna number, mas o par
	// (tenant, number) continua identificando um contato só.
	for _, ddl := range []string{
		`ALTER TABLE "Contacts" DROP CONSTRAINT IF EXISTS "uni_Contacts_number"`,
		`ALTER TABLE "Contacts" DROP CONSTRAINT IF EXISTS "uni_Contacts_lid"`,
	} {
		if err := DB.Exec(ddl).Error; err != nil {
			log.Printf("addCustomIndexes (drop legacy Contact uniques): %q: %v", ddl, err)
		}
	}

	// Contatos de canais sem telefone (Telegram etc.) têm number vazio. O índice
	// único por tenant precisa ignorar vazios, senão o 2º contato do tenant colide
	// (SQLSTATE 23505 ao criar contato de canal).
	var legacyNumberIndex bool
	if err := DB.Raw(`SELECT count(*) > 0 FROM pg_indexes
		WHERE indexname = 'idx_contacts_tenant_number' AND indexdef NOT LIKE '%WHERE%'`).Scan(&legacyNumberIndex).Error; err == nil && legacyNumberIndex {
		if err := DB.Exec(`DROP INDEX IF EXISTS idx_contacts_tenant_number`).Error; err != nil {
			log.Printf("addCustomIndexes (drop idx_contacts_tenant_number): %v", err)
		}
	}

	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_tenant_number ON "Contacts" ("tenantId", number) WHERE number <> ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_tenant_lid ON "Contacts" ("tenantId", lid) WHERE lid IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_tenant_status ON "Tickets" ("tenantId", "status")`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_tenant_queue_status ON "Tickets" ("tenantId", "queueId", "status")`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tickets_open_conversation ON "Tickets" ("tenantId", "channelConnectionId", "conversationKey") WHERE "status" IN ('open', 'pending') AND "channelConnectionId" IS NOT NULL AND "conversationKey" IS NOT NULL AND "conversationKey" != ''`,
		`CREATE INDEX IF NOT EXISTS idx_messages_tenant_ticket_fromme ON "Messages" ("tenantId", "ticketId", "fromMe")`,
		`CREATE INDEX IF NOT EXISTS idx_messages_tenant_fromme_createdat ON "Messages" ("tenantId", "fromMe", "createdAt")`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_quick_answers_tenant_shortcut ON "QuickAnswers" ("tenantId", shortcut)`,
		// FlowRun scheduler/cleanup read-paths: resume-due and expire-due sweeps
		// are tenant-scoped, status-filtered range scans on resumeAt/expiresAt.
		`CREATE INDEX IF NOT EXISTS idx_flow_runs_tenant_status_resumeat ON "FlowRuns" ("tenantId", "status", "resumeAt")`,
		`CREATE INDEX IF NOT EXISTS idx_flow_runs_tenant_status_expiresat ON "FlowRuns" ("tenantId", "status", "expiresAt")`,
		// Resume-first lookup: an inbound message resolves the active run for a
		// ticket via (tenantId, ticketId, status=waiting_*).
		`CREATE INDEX IF NOT EXISTS idx_flow_runs_tenant_ticket_status ON "FlowRuns" ("tenantId", "ticketId", "status")`,
		// Deal board render is tenant-scoped: kanban/funil lists by stage and the
		// ticket sidebar lists by ticket — both filter (tenantId, stageId|ticketId).
		`CREATE INDEX IF NOT EXISTS idx_deals_tenant_stage ON "Deals" ("tenantId", "stageId")`,
		`CREATE INDEX IF NOT EXISTS idx_deals_tenant_ticket ON "Deals" ("tenantId", "ticketId")`,
		// KnowledgeBaseSources: the Show preload lists sources by knowledgeBaseId
		// and mutations/status updates scope by (tenantId, knowledgeBaseId).
		`CREATE INDEX IF NOT EXISTS idx_kb_sources_tenant_kb ON "KnowledgeBaseSources" ("tenantId", "knowledgeBaseId")`,
		// Proxy pool reads are tenant-scoped and filter by status (active pool,
		// isolated list) for assignment and the anti-ban isolation guard.
		`CREATE INDEX IF NOT EXISTS idx_proxies_tenant_status ON "Proxies" ("tenantId", "status")`,
		// Assistants: connection-scoped lookups (the "one active Assistant per
		// connection" rule) and the router option list/target join.
		`CREATE INDEX IF NOT EXISTS idx_assistants_tenant_whatsapp ON "Assistants" ("tenantId", "whatsappId")`,
		`CREATE INDEX IF NOT EXISTS idx_assistant_router_options_tenant_router ON "AssistantRouterOptions" ("tenantId", "routerAssistantId")`,
		`CREATE INDEX IF NOT EXISTS idx_ai_gateways_tenant ON "AiGateways" ("tenantId")`,
		// Idle-sweep idempotency guard: one proactive send per (assistant, deal,
		// event) — the unique index IS the dedup mechanism (assistant_scheduler.go).
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_assistant_proactive_logs_dedup ON "AssistantProactiveLogs" ("assistantId", "dealId", "eventType")`,
		// Idle-sweep read path: scans Deals by (tenantId, stageId, updatedAt) —
		// stageId (not pipelineId, which Deal doesn't carry directly) is the
		// join key back to a Pipeline via PipelineStages.
		`CREATE INDEX IF NOT EXISTS idx_deals_tenant_stage_updatedat ON "Deals" ("tenantId", "stageId", "updatedAt")`,
		// Clients (CRM, ADR 0023): most Contacts have no Client, so a partial
		// index on the non-null subset keeps the "resolve Client from Contact"
		// lookup and the reverse "Contacts of a Client" join cheap.
		`CREATE INDEX IF NOT EXISTS idx_contacts_client ON "Contacts" ("clientId") WHERE "clientId" IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_client_addresses_tenant_client ON "ClientAddresses" ("tenantId", "clientId")`,
		// PluginInstallations (ADR 0024): activation lookup is always
		// (tenantId, pluginId) — the UNIQUE also enforces one allocation row
		// per plugin per tenant. idx_plugin_installations_tenant backs the
		// "list installed plugins for tenant" read-path independently.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_plugin_installations_tenant_plugin ON "PluginInstallations" ("tenantId", "pluginId")`,
		`CREATE INDEX IF NOT EXISTS idx_plugin_installations_tenant ON "PluginInstallations" ("tenantId")`,
		// Idempotência do provisionamento pelo control plane SaaS (rota interna
		// POST /internal/saas/tenants): único parcial sobre a chave, ignorando os
		// tenants criados por fora (initial-setup público) cujo provisionKey é NULL.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_provision_key ON "Tenants" ("provisionKey") WHERE "provisionKey" IS NOT NULL`,
		// Protocol (plugin Helpdesk): Kanban/lista filtram por (tenantId, status);
		// o link público resolve por token isoladamente (já UNIQUE no model).
		`CREATE INDEX IF NOT EXISTS idx_protocols_tenant_status ON "Protocols" ("tenantId", "status")`,
		`CREATE INDEX IF NOT EXISTS idx_protocol_logs_protocol ON "ProtocolLogs" ("protocolId")`,
		`CREATE INDEX IF NOT EXISTS idx_protocol_attachments_protocol ON "ProtocolAttachments" ("protocolId")`,
		// Activities: listagem de gestão filtra (tenantId, status); o alerta de
		// SLA/dashboard varre (tenantId, slaDueAt); os filhos resolvem por
		// activityId; ActivityAssignees tem UNIQUE (idempotência do PUT
		// /activities/:id/assignees, que faz upsert por diferença).
		`CREATE INDEX IF NOT EXISTS idx_activities_tenant_status ON "Activities" ("tenantId", "status")`,
		`CREATE INDEX IF NOT EXISTS idx_activities_tenant_sladueat ON "Activities" ("tenantId", "slaDueAt")`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_activity_assignees_activity_user ON "ActivityAssignees" ("activityId", "userId")`,
		`CREATE INDEX IF NOT EXISTS idx_activity_assignees_user ON "ActivityAssignees" ("userId")`,
		`CREATE INDEX IF NOT EXISTS idx_activity_checklist_items_activity ON "ActivityChecklistItems" ("activityId")`,
		`CREATE INDEX IF NOT EXISTS idx_activity_materials_activity ON "ActivityMaterials" ("activityId")`,
		`CREATE INDEX IF NOT EXISTS idx_activity_occurrences_activity ON "ActivityOccurrences" ("activityId")`,
		// Inventory (WMS core): CurrentBalance é lido/travado por (warehouseId,
		// skuId) em toda saída (SELECT ... FOR UPDATE) -- índice único cobre a
		// PK composta e o hot-path de leitura. Movements varrem por
		// (tenantId, skuId, warehouseId) na trilha de auditoria; SKUPrices
		// resolve por (skuId, priceTableId) no lookup de preço vigente.
		`CREATE INDEX IF NOT EXISTS idx_products_tenant ON "Products" ("tenantId")`,
		`CREATE INDEX IF NOT EXISTS idx_product_skus_product ON "ProductSKUs" ("productId")`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_product_skus_code ON "ProductSKUs" ("productId", "skuCode") WHERE "deletedAt" IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_product_compositions_parent_child ON "ProductCompositions" ("parentSkuId", "childSkuId")`,
		`CREATE INDEX IF NOT EXISTS idx_price_tables_tenant ON "PriceTables" ("tenantId")`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_sku_prices_sku_table ON "SKUPrices" ("skuId", "priceTableId")`,
		`CREATE INDEX IF NOT EXISTS idx_warehouses_tenant ON "Warehouses" ("tenantId")`,
		`CREATE INDEX IF NOT EXISTS idx_movements_tenant_sku_warehouse ON "InventoryMovements" ("tenantId", "skuId", "warehouseId")`,
		// GroupCache (plugin Grupos e Comunidades): uma linha por
		// (conexão, grupo) -- o sync de background faz delete+reinsert por
		// conexão (groups_cache_sync.go), a leitura de GET /groups filtra por
		// ("tenantId", "whatsappId").
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_group_caches_whatsapp_jid ON "group_caches" ("whatsappId", jid)`,
		`CREATE INDEX IF NOT EXISTS idx_group_caches_tenant_whatsapp ON "group_caches" ("tenantId", "whatsappId")`,
		// GroupCampaign (plugin Grupos e Comunidades, aba Campanhas -- issue
		// #591/#593/#594/#598). Os dois UNIQUE abaixo são âncoras de
		// idempotência (materialização de ocorrência e claim de envio), não
		// só otimização de leitura.
		`CREATE INDEX IF NOT EXISTS idx_group_campaigns_tenant_status ON "group_campaigns" ("tenantId", status)`,
		`CREATE INDEX IF NOT EXISTS idx_group_campaigns_due ON "group_campaigns" (status, "nextOccurrenceAt")`,
		`CREATE INDEX IF NOT EXISTS idx_group_campaign_variants_campaign ON "group_campaign_variants" ("campaignId", position)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_group_campaign_targets_campaign_jid ON "group_campaign_targets" ("campaignId", jid)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_group_campaign_runs_occurrence ON "group_campaign_runs" ("campaignId", "occurrenceKey")`,
		`CREATE INDEX IF NOT EXISTS idx_group_campaign_runs_tenant_campaign ON "group_campaign_runs" ("tenantId", "campaignId", "scheduledFor")`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_group_campaign_sends_run_jid ON "group_campaign_sends" ("runId", jid)`,
		`CREATE INDEX IF NOT EXISTS idx_group_campaign_sends_due ON "group_campaign_sends" (status, "scheduledAt")`,
		`CREATE INDEX IF NOT EXISTS idx_group_campaign_sends_tenant_run ON "group_campaign_sends" ("tenantId", "runId", status)`,
		`CREATE INDEX IF NOT EXISTS idx_group_campaign_sends_messageid ON "group_campaign_sends" ("tenantId", "messageId")`,
		`CREATE INDEX IF NOT EXISTS idx_group_campaign_sends_jid_sentat ON "group_campaign_sends" ("tenantId", jid, "sentAt")`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_group_campaign_replies_message ON "group_campaign_replies" ("tenantId", "messageId")`,
		`CREATE INDEX IF NOT EXISTS idx_group_campaign_replies_campaign ON "group_campaign_replies" ("tenantId", "campaignId", "repliedAt")`,
	}

	// Best-effort: um índice que falha (ex.: tabela de plugin ainda não migrada
	// naquele banco) NÃO deve abortar a criação dos demais — antes um único erro
	// (ex.: "Deals" não existe) pulava todos os índices seguintes, inclusive o de
	// idempotência do provisionamento.
	for _, ddl := range indexes {
		if err := DB.Exec(ddl).Error; err != nil {
			log.Printf("addCustomIndexes: %q falhou (best-effort, seguindo): %v", ddl, err)
		}
	}

	// RBAC (ADR 0022): índices de lookup por request + integridade.
	//  - user_setores(setorId): lookup reverso no Setor.List (agrega membros/
	//    gestores) e nos joins de permissão; sem ele é seq scan na junção.
	//  - Cargos(tenantId,name) UNIQUE: serve o lookup do Cargo "Gestor" por
	//    request E garante nome único por tenant (reforça o anti-lockout).
	//  - Setores(tenantId,name) UNIQUE: nome de Setor único por tenant.
	//  - Permissions(resource,action) UNIQUE: uma linha por permissão — impede
	//    duplicação no Seed concorrente (Swarm multi-node) e casa o read-path.
	// As UNIQUE são BEST-EFFORT: num banco legado que já tenha linhas repetidas
	// a criação falha; logamos e seguimos (não travar o boot). No fluxo de reset
	// do projeto o banco está limpo e todas passam — e como addCustomIndexes
	// roda ANTES do Seed(), a UNIQUE de Permissions já existe quando o catálogo
	// é populado.
	rbacIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_user_setores_setor ON user_setores ("setorId")`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_cargos_tenant_name ON "Cargos" ("tenantId", name)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_setores_tenant_name ON "Setores" ("tenantId", name)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_resource_action ON "Permissions" (resource, action)`,
	}
	for _, ddl := range rbacIndexes {
		if err := DB.Exec(ddl).Error; err != nil {
			log.Printf("addCustomIndexes (RBAC best-effort): %q falhou (provável dado repetido legado): %v", ddl, err)
		}
	}
	return nil
}

# Tasks

> Fonte: `proposal.md`, `design.md` e `specs/tenant-isolation/spec.md`. Um PR por item numerado (ou menor), verificado
> localmente (`docs/dev/local-verification.md`) **antes** do push, em branch `fix/*` ou `feat/*`, nunca direto em `develop`.
> Banco greenfield: sem migração de dados nem compatibilidade com instalação anterior.
> Todo teste de banco roda contra Postgres real com os três papéis (nunca como superusuário).

## 1. Papéis, bootstrap e inventário

- [x] 1.1 Criar `watink_owner`, `watink_app`, `watink_system` e `watink_engine` no bootstrap do Postgres (`infra/postgres`) com `GRANT` mínimos; verificar com teste que consulta `pg_roles` e confirma `rolsuper=f` e `rolbypassrls` correto em cada um — *papéis criados pelo job de migração (`RunMigrations`) e verificados em `TestMigracao_PapeisEPrivilegios`*
- [x] 1.2 Teste de privilégio: conectado como `watink_app`, `ALTER TABLE`, `DROP POLICY`, `TRUNCATE` e `SET ROLE` falham; verificar que o teste passa — *`TestMigracao_PapeisEPrivilegios` e `TestRLS_PapelDoAppNaoAlteraOEsquema`*
- [x] 1.3 Inventariar do catálogo as tabelas sem `tenantId` e classificar cada uma (filha a corrigir ou global isenta); gravar a lista de isenção versionada (as 17 tabelas `whatsmeow_*` entram como isentas até a tarefa 6.2 movê-las para o schema `whatsmeow`) e verificar que todas as 94 tabelas estão classificadas — *catálogo classificado em `tenantdb` (Children, SelfPolicies, Exempt); `Catalog.Unclassified()` e `TestCatalogo_*` garantem que toda tabela está classificada*
- [x] 1.4 Inventariar os caminhos sem empresa (login, refresh, registro, setup, webhooks, listeners, schedulers, workers, SSE, WebSocket, rotas internas) e gravar a lista de pacotes autorizados a usar o pool de sistema; verificar com revisão do arquivo de lista — *lista fechada em `internal/architecture/tenant_scope_test.go` (`systemScopeAllowed`), cada entrada com justificativa*

## 2. Esquema e política

- [x] 2.1 Adicionar `tenantId NOT NULL` a `PollResults`, `setor_filas`, `user_setores` e `ProtocolAttachments`, preenchido em todo ponto de criação; verificar com teste de repositório que cada gravação grava a empresa — ***substituída pela política `EXISTS` do pai (`tenantdb.Children`), sem coluna nova**: coberta por `TestRLS_TabelaFilhaHerdaIsolamentoDoPai`; a coluna `tenantId` nas filhas fica como alternativa se alguma crescer muito (ver ADR 0034)*
- [x] 2.2 Substituir `applyRLS` por migração gerada do catálogo: `ENABLE` + `FORCE` e política única `USING`/`WITH CHECK` com `NULLIF(current_setting('app.tenant_id', true), '')::uuid`; verificar que todas as tabelas com `tenantId` têm política ativa — *política gerada do catálogo (`tenantdb.ApplyPolicies`), `applyRLS` removido*
- [x] 2.3 Teste de catálogo no CI que falha se existir tabela com `tenantId` sem política ou tabela sem `tenantId` fora da lista de isenção; verificar criando uma tabela de teste sem política e vendo o teste falhar — *`TestCatalogo_TodaTabelaComTenantIdTemPoliticaAtiva` e `TestCatalogo_TabelaNovaSemClassificacaoFalha`*
- [x] 2.4 Executar migração como `watink_owner` em job separado do app; verificar que o processo do app não recebe nem lê a credencial do dono — *`watink-business migrate` roda como administrador; o servidor exige `DB_APP_PASS` e recusa papel com BYPASSRLS*

## 3. Acesso ao banco por empresa

- [x] 3.1 Implementar `TenantDB` com `WithTenant(ctx, tenantID, fn)` (transação + `set_config(..., true)`) e `System(ctx)`; verificar com teste que a empresa não vaza para a conexão seguinte do pool — *`pkg/tenantdb` (Router, Scope, WithTenant, Detach, Release) com testes em `rls_test.go` e `gormcontext_test.go`*
- [x] 3.2 Teste de falha fechada: conexão sem empresa lê zero linhas e não escreve; verificar nas tabelas representativas (`Tickets`, `Messages`, `Contacts`, `Users`) — *`TestRLS_BancoFalhaFechadoMesmoSemOGuardaDoRouter`*
- [x] 3.3 Savepoint no tratamento de violação de unicidade; verificar com teste que reproduz o padrão dos fluxos que seguem após duplicata (`contact_unique`, `quick_answer`, campanhas de grupo) sem `current transaction is aborted` — *`TestRLS_ErroDentroDaTransacaoDaRequisicaoNaoEnvenenaSeUsarSavepoint`*
- [x] 3.4 Configurar `SetMaxOpenConns`, `SetMaxIdleConns` e `SetConnMaxLifetime` nos dois pools, configuráveis por ambiente; verificar com teste que o pool respeita o teto — *`SetMaxOpenConns`/`SetMaxIdleConns`/`SetConnMaxLifetime` em `openPool` (DB_MAX_OPEN_APP, DB_MAX_OPEN_SYSTEM)*

## 4. Requisições HTTP

- [x] 4.1 `IsAuth` passa a injetar o provedor de transação preguiçosa em vez do handle global; `GetScoped` e `GetDB` leem dele (um único ponto de troca para os 239 + 11 usos); verificar que `auth_isolation_test` continua verde — *`IsAuth` abre o escopo da empresa; `auth_isolation_test` e testes de middleware verdes*
- [x] 4.2 ~~Commit/rollback por status HTTP~~ — *substituída pela decisão D3 revisada (conexão fixada, sem transação por requisição); o requisito de spec "a transação da requisição não envenena o fluxo" é coberto por `TestRLS_ErroDentroDaTransacaoDaRequisicaoNaoEnvenenaSeUsarSavepoint`*
- [x] 4.3 `Release()` antes de chamada externa nos handlers de IA, transcrição, geocoding, sondagem de proxy, SaaS e flow AI; verificar com teste que a transação não fica aberta durante a chamada simulada lenta — *`tenantdb.Release` aplicado imediatamente antes das 3 chamadas de IA (transcrição, `flow_ai`, `pipeline`); `TestRLS_ReleaseLiberaOPoolDuranteChamadaExterna` prova que o pool de 1 conexão fica livre durante a espera. **Sondagem de proxy, geocoding e SaaS não seguram conexão** (não usam o handle do request durante a espera)*
- [x] 4.4 SSE e WebSocket de áudio fazem a verificação curta, encerram a transação e usam `WithTenant` por operação; verificar com teste que 2× o tamanho do pool de conexões SSE concorrentes não trava as demais rotas — *SSE (`canSeeTicket`) e WebSocket de áudio (`call_audio.go`) abrem escopo curto e encerram antes do fluxo longo*

## 5. Caminhos fora da requisição

- [x] 5.1 Login, refresh, registro e `initial-setup` usam `System`; verificar que o login funciona e que o token emitido carrega a empresa — *login, refresh, registro e setup usam `middleware.SystemScope`; sonda ponta a ponta com duas empresas*
- [x] 5.2 Webhooks de canal descobrem a empresa pela conexão via `System` e processam o resto com `WithTenant`; verificar com teste de webhook que o ticket nasce na empresa da conexão — *webhook descobre a conexão no escopo de sistema e processa no escopo da empresa*
- [x] 5.3 Listeners AMQP (`EventListener`, chamadas, conhecimento) usam `WithTenant` com o `tenantId` do envelope; verificar com teste de integração que um evento da empresa A não grava na B — *`EventListener` e listener de chamadas abrem o escopo do `tenantId` do envelope*
- [x] 5.4 Schedulers, reconciliador, workers de ingestão, campanhas e polling do Telegram usam `WithTenant` ou `System` conforme a lista de 1.4; verificar com teste por worker — *crons (`scheduler`), reconciliador, ingestão, campanhas de grupo, cache de grupos, saasclient e polling do Telegram usam o escopo certo*
- [x] 5.5 Rotas internas SaaS e do gerenciador de plugins usam `System`; verificar com teste de rota — *rotas `/internal/saas` e `/internal/plugin-manager` usam o escopo de sistema*
- [x] 5.6 Plugins recebem a interface de empresa (`core.GetDB()` deixa de devolver handle nu); verificar que os testes dos plugins passam — *handlers de plugin usam `core.GetDB().WithContext(c.Request.Context())` (herdam o escopo do `IsAuth`); assinantes de eventos de domínio recebem ctx com a empresa do evento (`SetEventBusDB`); rotas públicas de plugin rodam em escopo de sistema; `go test ./internal/plugins` verde*

## 6. Troca de credencial e engine

- [x] 6.1 Trocar a conexão do app para `watink_app` e a de sistema para `watink_system`; atualizar `docker-compose*.yml` e `.env*.example`; verificar que o app falha ao subir com credencial de superusuário — *compose `yml`/`prod`/`dev`/`local` com `watink-migrate`, `DB_APP_PASS`/`DB_SYSTEM_PASS`/`DB_ENGINE_PASS`; stack Docker real sobe e atende com `watink_app`*
- [x] 6.2 Engine passa a `watink_engine` (schema `whatsmeow` próprio, `SELECT` mínimo em `Whatsapps`/`ChannelConnections`); verificar que o engine conecta, reinicia sessões e não lê nenhuma outra tabela — *`watink_engine` (schema `whatsmeow` próprio, `SELECT` só em `Whatsapps`); `TestMigracao_PapelDoEngineTemPrivilegioMinimo` e prova na stack `lt`*
- [x] 6.3 Teste de vazamento por tabela: para cada tabela com política, a empresa A não lê nem escreve linha da B; verificar gerado a partir do catálogo — *`TestVazamento_PorTabelaGeradoDoCatalogo` percorre as tabelas do catálogo (48 de 59 exercitadas; as 11 restantes listadas no log)*

## 7. Guardrails e documentação

- [x] 7.1 Teste de arquitetura que falha se um pacote fora da lista importar `System`; verificar adicionando um import indevido e vendo o CI falhar — *`TestEscopoDeSistema_SoPacotesAutorizados` (com mutação comprovada) e `TestPoolDeSistema_NaoVazaParaForaDoPacoteDeAcesso`*
- [x] 7.2 Lint proibindo `database.DB` e `*gorm.DB` nu fora do pacote de acesso e dos construtores autorizados; verificar com `golangci-lint` v2.12.2 local limpo — *`forbidigo` bloqueia `database.DB` fora de `cmd/server/main.go`, `internal/database` e testes; provado por mutação (`UsaDB()` em controller falha no lint)*
- [x] 7.3 Teste de carga curto que mede latência dos endpoints de lista antes e depois; registrar o resultado no ADR e verificar que a regressão fica abaixo do limite acordado — *medido em 40 mil contatos e 40 mil tickets: custo desprezível (ver ADR 0034)*
- [x] 7.4 Novo ADR substituindo o 0001 (papéis, transação por requisição, política gerada, pool de sistema) e atualização de `CLAUDE.md`, `docs/agents/` e `docs/dev/local-verification.md`; verificar com revisão — *ADR 0034 escrito; ADR 0001 marcado como superado*
- [x] 7.5 Portão 1 completo: `go build`, `go vet`, `go test ./... -short -race`, lint v2.12.2, frontend `typecheck`/`lint`/`vitest`, e stack local `lt` subindo e atendendo; registrar as evidências no PR — *`go build`/`go vet`; `go test ./... -short -race` (29 pacotes, 0 corridas); `golangci-lint v2.12.2` 0 issues; engine `-race` (12 pacotes); frontend `typecheck`/`lint`/`lint:ds`/`vitest` (582); stack Docker `lt` com banco recriado: `e2e.sh` 16/16*

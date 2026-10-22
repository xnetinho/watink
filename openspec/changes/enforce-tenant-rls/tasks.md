# Tasks

> Fonte: `proposal.md`, `design.md` e `specs/tenant-isolation/spec.md`. Um PR por item numerado (ou menor), verificado
> localmente (`docs/dev/local-verification.md`) **antes** do push, em branch `fix/*` ou `feat/*`, nunca direto em `develop`.
> Banco greenfield: sem migração de dados nem compatibilidade com instalação anterior.
> Todo teste de banco roda contra Postgres real com os três papéis (nunca como superusuário).

## 1. Papéis, bootstrap e inventário

- [ ] 1.1 Criar `watink_owner`, `watink_app`, `watink_system` e `watink_engine` no bootstrap do Postgres (`infra/postgres`) com `GRANT` mínimos; verificar com teste que consulta `pg_roles` e confirma `rolsuper=f` e `rolbypassrls` correto em cada um
- [ ] 1.2 Teste de privilégio: conectado como `watink_app`, `ALTER TABLE`, `DROP POLICY`, `TRUNCATE` e `SET ROLE` falham; verificar que o teste passa
- [ ] 1.3 Inventariar do catálogo as tabelas sem `tenantId` e classificar cada uma (filha a corrigir ou global isenta); gravar a lista de isenção versionada (as 17 tabelas `whatsmeow_*` entram como isentas até a tarefa 6.2 movê-las para o schema `whatsmeow`) e verificar que todas as 94 tabelas estão classificadas
- [ ] 1.4 Inventariar os caminhos sem empresa (login, refresh, registro, setup, webhooks, listeners, schedulers, workers, SSE, WebSocket, rotas internas) e gravar a lista de pacotes autorizados a usar o pool de sistema; verificar com revisão do arquivo de lista

## 2. Esquema e política

- [ ] 2.1 Adicionar `tenantId NOT NULL` a `PollResults`, `setor_filas`, `user_setores` e `ProtocolAttachments`, preenchido em todo ponto de criação; verificar com teste de repositório que cada gravação grava a empresa
- [ ] 2.2 Substituir `applyRLS` por migração gerada do catálogo: `ENABLE` + `FORCE` e política única `USING`/`WITH CHECK` com `NULLIF(current_setting('app.tenant_id', true), '')::uuid`; verificar que todas as tabelas com `tenantId` têm política ativa
- [ ] 2.3 Teste de catálogo no CI que falha se existir tabela com `tenantId` sem política ou tabela sem `tenantId` fora da lista de isenção; verificar criando uma tabela de teste sem política e vendo o teste falhar
- [ ] 2.4 Executar migração como `watink_owner` em job separado do app; verificar que o processo do app não recebe nem lê a credencial do dono

## 3. Acesso ao banco por empresa

- [ ] 3.1 Implementar `TenantDB` com `WithTenant(ctx, tenantID, fn)` (transação + `set_config(..., true)`) e `System(ctx)`; verificar com teste que a empresa não vaza para a conexão seguinte do pool
- [ ] 3.2 Teste de falha fechada: conexão sem empresa lê zero linhas e não escreve; verificar nas tabelas representativas (`Tickets`, `Messages`, `Contacts`, `Users`)
- [ ] 3.3 Savepoint no tratamento de violação de unicidade; verificar com teste que reproduz o padrão dos fluxos que seguem após duplicata (`contact_unique`, `quick_answer`, campanhas de grupo) sem `current transaction is aborted`
- [ ] 3.4 Configurar `SetMaxOpenConns`, `SetMaxIdleConns` e `SetConnMaxLifetime` nos dois pools, configuráveis por ambiente; verificar com teste que o pool respeita o teto

## 4. Requisições HTTP

- [ ] 4.1 `IsAuth` passa a injetar o provedor de transação preguiçosa em vez do handle global; `GetScoped` e `GetDB` leem dele (um único ponto de troca para os 239 + 11 usos); verificar que `auth_isolation_test` continua verde
- [ ] 4.2 Commit em resposta 2xx/3xx, rollback em 4xx/5xx e em panic; verificar com teste de middleware que escrita some em resposta de erro
- [ ] 4.3 `Release()` antes de chamada externa nos handlers de IA, transcrição, geocoding, sondagem de proxy, SaaS e flow AI; verificar com teste que a transação não fica aberta durante a chamada simulada lenta
- [ ] 4.4 SSE e WebSocket de áudio fazem a verificação curta, encerram a transação e usam `WithTenant` por operação; verificar com teste que 2× o tamanho do pool de conexões SSE concorrentes não trava as demais rotas

## 5. Caminhos fora da requisição

- [ ] 5.1 Login, refresh, registro e `initial-setup` usam `System`; verificar que o login funciona e que o token emitido carrega a empresa
- [ ] 5.2 Webhooks de canal descobrem a empresa pela conexão via `System` e processam o resto com `WithTenant`; verificar com teste de webhook que o ticket nasce na empresa da conexão
- [ ] 5.3 Listeners AMQP (`EventListener`, chamadas, conhecimento) usam `WithTenant` com o `tenantId` do envelope; verificar com teste de integração que um evento da empresa A não grava na B
- [ ] 5.4 Schedulers, reconciliador, workers de ingestão, campanhas e polling do Telegram usam `WithTenant` ou `System` conforme a lista de 1.4; verificar com teste por worker
- [ ] 5.5 Rotas internas SaaS e do gerenciador de plugins usam `System`; verificar com teste de rota
- [ ] 5.6 Plugins recebem a interface de empresa (`core.GetDB()` deixa de devolver handle nu); verificar que os testes dos plugins passam

## 6. Troca de credencial e engine

- [ ] 6.1 Trocar a conexão do app para `watink_app` e a de sistema para `watink_system`; atualizar `docker-compose*.yml` e `.env*.example`; verificar que o app falha ao subir com credencial de superusuário
- [ ] 6.2 Engine passa a `watink_engine` (schema `whatsmeow` próprio, `SELECT` mínimo em `Whatsapps`/`ChannelConnections`); verificar que o engine conecta, reinicia sessões e não lê nenhuma outra tabela
- [ ] 6.3 Teste de vazamento por tabela: para cada tabela com política, a empresa A não lê nem escreve linha da B; verificar gerado a partir do catálogo

## 7. Guardrails e documentação

- [ ] 7.1 Teste de arquitetura que falha se um pacote fora da lista importar `System`; verificar adicionando um import indevido e vendo o CI falhar
- [ ] 7.2 Lint proibindo `database.DB` e `*gorm.DB` nu fora do pacote de acesso e dos construtores autorizados; verificar com `golangci-lint` v2.12.2 local limpo
- [ ] 7.3 Teste de carga curto que mede latência dos endpoints de lista antes e depois; registrar o resultado no ADR e verificar que a regressão fica abaixo do limite acordado
- [ ] 7.4 Novo ADR substituindo o 0001 (papéis, transação por requisição, política gerada, pool de sistema) e atualização de `CLAUDE.md`, `docs/agents/` e `docs/dev/local-verification.md`; verificar com revisão
- [ ] 7.5 Portão 1 completo: `go build`, `go vet`, `go test ./... -short -race`, lint v2.12.2, frontend `typecheck`/`lint`/`vitest`, e stack local `lt` subindo e atendendo; registrar as evidências no PR

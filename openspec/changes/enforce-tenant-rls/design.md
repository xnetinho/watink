# Design

## Context

Estado atual, medido no código e no banco local:

- O app e o engine conectam como `postgres` (`rolsuper=t`, `rolbypassrls=t`); `BYPASSRLS` ignora políticas mesmo com
  `FORCE ROW LEVEL SECURITY`.
- 59 tabelas têm `tenantId`; só 10 têm política (`applyRLS`), e a política lê `app.current_tenant`, que ninguém define.
- `IsAuth` injeta o `*gorm.DB` global no contexto (`c.Set("db", db)`); `auth.GetScoped` devolve `db.Where("tenantId" = ?)`.
  Não existe transação por requisição.
- Há 31 chamadas a `db.Transaction(...)`/`Begin()` e 14 pontos que tratam violação de unicidade como fluxo normal.
- Fora de requisição, o handle global é capturado por construtores (44 `New...(db *gorm.DB)`) e usado por listeners AMQP,
  schedulers, workers e goroutines.
- Quatro tabelas filhas não têm `tenantId`: `PollResults`, `setor_filas`, `user_setores`, `ProtocolAttachments`.
- Não há `SetMaxOpenConns`: o pool não tem teto.

## Goals / Non-Goals

**Goals:**
- O banco recusa leitura e escrita entre empresas sem depender de filtro da aplicação.
- Falhar fechado: sem empresa definida, zero linhas.
- Nenhuma tabela com `tenantId` fica sem política, garantido por teste.
- Caminhos sem empresa são poucos, nomeados e auditáveis.

**Non-Goals:**
- Não remove os filtros `"tenantId"` existentes. Continuam como defesa em profundidade e por desempenho (índice).
- Não migra dados nem preserva instalações antigas: o banco é recriado do zero.
- Não troca o ORM nem a estrutura de controllers (isso é a onda de arquitetura).
- Não introduz schema por empresa nem banco por empresa.

## Decisions

### D1. Três papéis, dono separado
`watink_owner` é dono dos objetos e roda apenas migrações. `watink_app` é `NOSUPERUSER NOBYPASSRLS`, com `SELECT/INSERT/UPDATE/DELETE`
e `USAGE` em sequências, e atende toda requisição. `watink_system` é `NOSUPERUSER BYPASSRLS`, só DML, e é um **segundo pool**
usado apenas por caminhos sem empresa.
*Por quê:* `FORCE ROW LEVEL SECURITY` só vale contra o dono; o dono não pode ser o papel do app, senão ele desligaria a
política. *Alternativa rejeitada:* um papel só com `FORCE`; o dono ainda pode `ALTER TABLE ... DISABLE ROW LEVEL SECURITY`.

### D2. Valor de empresa por transação, com `set_config(..., true)`
Cada acesso com empresa roda numa transação que executa `SELECT set_config('app.tenant_id', $1, true)` primeiro. O terceiro
argumento `true` limita o valor à transação, então commit ou rollback o apagam e a conexão volta limpa ao pool.
*Por quê:* é a única forma segura com pool; `SET` de sessão vazaria o valor para a próxima requisição que reutilizar a conexão.
*Alternativa rejeitada:* uma conexão dedicada por requisição com `SET` de sessão; esgota o pool e vaza se faltar o reset.

### D3. Transação preguiçosa por requisição, com savepoints
O middleware não abre transação logo no início. Ele coloca no contexto um provedor que, na **primeira** consulta, abre a
transação, define a empresa e a reutiliza até o fim; `commit` em resposta 2xx/3xx, `rollback` em 4xx/5xx ou panic.
Chamadas a `db.Transaction` dentro dela viram **savepoint** automaticamente (comportamento do GORM sobre `*gorm.DB` já em
transação), e o tratamento de violação de unicidade usa savepoint (`SAVEPOINT`/`ROLLBACK TO`) para não abortar a transação inteira.
*Por quê:* abrir transação sem uso custa um round-trip por rota; e um erro de SQL numa transação a deixa abortada
(`current transaction is aborted`), o que quebraria os 14 fluxos que tratam duplicata e seguem.
*Alternativa rejeitada:* transação não preguiçosa; custo em toda rota, inclusive as que nem tocam o banco.

### D4. Transação não atravessa chamada externa
Handlers que chamam serviço externo (IA, geocoding, SaaS, transcrição, sondagem de proxy) não podem manter a transação
aberta durante a espera. O provedor expõe `Release()` para encerrar a transação antes da chamada, e um teste de política
falha se um handler da lista de chamadas externas a segurar. Streams longos (SSE, WebSocket de áudio) **não** usam a
transação da requisição: fazem uma verificação curta, a encerram e dali em diante usam `WithTenant` por operação.
*Por quê:* uma transação aberta durante 90 s de IA, ou durante uma chamada de voz, esgota o pool.

### D5. `WithTenant(ctx, db, tenantID, fn)` para tudo fora de requisição
Listeners AMQP, schedulers, workers e goroutines recebem o `tenantID` do evento (o envelope já o carrega) e executam
`fn(tx)` numa transação com a empresa definida. Construtores deixam de guardar um `*gorm.DB` utilizável sem empresa:
passam a receber uma interface `TenantDB` com `WithTenant` e `System`.
*Por quê:* torna impossível o acesso "esquecido": o tipo não oferece consulta sem empresa.

### D6. Pool de sistema com lista fechada
`System(ctx)` usa `watink_system`. Casos permitidos: login e refresh (descobrir a empresa pelo e-mail), registro e
`initial-setup`, descoberta da empresa de um webhook de canal pela conexão, varreduras globais de manutenção (leader lock),
rotas internas SaaS e do gerenciador de plugins, e migrações de seed. Um teste de arquitetura lista os pacotes
autorizados a importar `System`; qualquer outro falha o CI.

### D7. Política gerada do catálogo, não escrita tabela a tabela
Uma migração percorre `information_schema.columns` onde `column_name = 'tenantId'` e cria, em cada tabela:
`ENABLE` + `FORCE ROW LEVEL SECURITY` e uma política única
`USING ("tenantId" = NULLIF(current_setting('app.tenant_id', true), '')::uuid) WITH CHECK (<igual>)`.
`NULLIF` transforma "não definido" em `NULL`, e comparação com `NULL` nunca é verdadeira: falha fechada. Um teste lê o
catálogo e falha se alguma tabela com `tenantId` estiver sem política.
*Por quê:* um teste de catálogo pega o esquecimento que a lista manual (`applyRLS`, 10 tabelas) deixou passar.

### D8. Tabelas filhas ganham `tenantId`
`PollResults`, `setor_filas`, `user_setores` e `ProtocolAttachments` recebem `tenantId NOT NULL` preenchido na criação,
com o mesmo tratamento das demais. *Alternativa rejeitada:* política por `EXISTS` na tabela pai; mais lenta e mais fácil de errar.

### D9. Tabelas realmente globais ficam fora, por lista explícita
`Plans`, `Permissions`, `InstancePolicies` e afins não têm dono de empresa. Entram numa lista versionada de isenção, e o
teste de catálogo exige que toda tabela sem `tenantId` esteja nela.

### D10. Pool com teto e engine separado
`SetMaxOpenConns`, `SetMaxIdleConns` e `SetConnMaxLifetime` explícitos nos dois pools. O engine passa a um papel próprio
(`watink_engine`, só `SELECT` em `Whatsapps`/`ChannelConnections` e dono do schema `whatsmeow`), e a carga inicial de sessões
deve sair do SQL no futuro handshake (`consolidate-platform-foundation`).

### D11. Esquema e papéis criados no bootstrap, não no app
Os papéis e os `GRANT` são criados por script de bootstrap do Postgres (`infra/postgres`) e por um job de migração que roda
como `watink_owner`. O processo do app nunca recebe a credencial do dono.

## Risks / Trade-offs

- **Transação por requisição segura conexão do pool.** Mitigação: preguiçosa, `Release()` antes de chamada externa, teto de
  pool e teste de carga que prova que 2× o tamanho do pool de requisições concorrentes não trava.
- **Erro de SQL aborta a transação.** Mitigação: savepoint no tratamento de duplicata, e teste que reproduz o padrão dos 14 fluxos.
- **Caminhos esquecidos retornam zero linhas em silêncio** (falha fechada, mas funcional): um listener sem `WithTenant`
  deixaria de gravar. Mitigação: tipo `TenantDB` sem consulta nua, e teste de integração por caminho (listener, scheduler, webhook).
- **Desempenho.** `set_config` + política custam um round-trip por transação e um predicado por linha; o predicado usa o
  índice já existente em `tenantId`. Mitigação: medir antes e depois nos endpoints de lista.
- **Mudança ampla.** Toca 44 construtores. Mitigação: um PR por área (base, middleware, listeners, schedulers, plugins),
  cada um com teste de vazamento.
- **Credenciais novas.** O app não sobe sem os três papéis; `.env*.example` e compose mudam (**BREAKING**, aceito).

## Migration Plan

Banco greenfield: recriar do zero, sem retrocompatibilidade. Ordem de entrega: (1) papéis e política no bootstrap com o app
ainda no papel antigo (política inerte, nada muda); (2) `TenantDB`, middleware e `WithTenant`; (3) migrar caminhos fora de
requisição; (4) trocar a conexão do app para `watink_app`; (5) ligar o teste de catálogo e o lint do pool de sistema no CI.
Reverter = voltar a credencial do app ao papel anterior; as políticas permanecem inofensivas.

## Open Questions

- Nenhuma que bloqueie o início. A lista de isenção (D9) e a lista de pacotes autorizados a usar `System` (D6) serão
  fechadas na tarefa 1.3 com base no catálogo real.

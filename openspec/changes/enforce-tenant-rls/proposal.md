# Proposal

## Why

O isolamento entre empresas hoje depende só de um `WHERE "tenantId"` escrito à mão em cada consulta (ADR 0001). O RLS
nunca funcionou: o app e o engine conectam como `postgres` (superusuário, `BYPASSRLS`), só 10 das 59 tabelas com
`tenantId` têm política, e a aplicação não abre transação por requisição, então nem existe onde guardar o tenant.
Um filtro esquecido em uma query nova vaza dados de outra empresa sem nenhum erro. Como não há instância em produção,
este é o momento de trocar a disciplina por um mecanismo que o banco impõe.

## What Changes

- **Três papéis de banco** com privilégio mínimo: `watink_owner` (dono dos objetos, só migrações), `watink_app`
  (`NOSUPERUSER NOBYPASSRLS`, só DML, usado por toda requisição) e `watink_system` (`BYPASSRLS`, só DML, usado por um
  segundo pool restrito a caminhos explicitamente sem tenant).
- **Transação por requisição, preguiçosa**, com `set_config('app.tenant_id', ..., true)` local à transação: o escopo
  some sozinho no commit ou rollback (falha fechada).
- **Política RLS em todas as tabelas com `tenantId`**, gerada a partir do catálogo, com `USING` e `WITH CHECK`; sem
  tenant definido, nenhuma linha é visível nem gravável.
- **Tabelas filhas ganham `tenantId`** (`PollResults`, `setor_filas`, `user_setores`, `ProtocolAttachments`).
- **Caminhos sem requisição** (login, webhooks, listeners AMQP, schedulers, workers, SSE, WebSocket de áudio) passam a
  usar `WithTenant(ctx, tenantID, fn)` ou o pool de sistema, nunca o handle global.
- **Pool de conexões** com limites explícitos.
- **Engine** deixa de usar `postgres` e passa a um papel com privilégio mínimo.
- **Guardrails:** teste de catálogo que falha se uma tabela com `tenantId` não tiver política, teste de vazamento por
  tabela e lint que restringe o pool de sistema a uma lista de pacotes.
- **BREAKING:** variáveis de ambiente novas para os três papéis; o app não sobe com credencial de superusuário.

## Capabilities

### New Capabilities
- `tenant-isolation`: isolamento de dados entre empresas imposto pelo banco, independente dos filtros da aplicação.

### Modified Capabilities

## Impact

- `business/pkg/auth`, `business/internal/middleware`, `business/internal/database`, `business/cmd/server/main.go`.
- Todos os pontos que usam `database.DB` ou capturam `c.Get("db")` fora da requisição (13 usos diretos, listeners,
  schedulers, workers, goroutines).
- `engine-go` (`session_loader.go`, DSN), `docker-compose*.yml`, `infra/postgres/init.sql`, `.env*.example`.
- ADR 0001 é substituído por um novo ADR.

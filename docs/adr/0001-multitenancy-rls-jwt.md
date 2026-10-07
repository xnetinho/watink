# Multitenancy: isolamento por `"tenantId"` explícito (RLS inerte) + JWT

Isolamento de dados entre Tenants. O `tenantId` vem do JWT (claim validada como UUID no `IsAuth`) e **todo acesso
filtra por `"tenantId"` de forma explícita**: `auth.GetScoped(c, "<Tabela>")` nos controllers (232 pontos) e
`WHERE "tenantId"` manual nos workers, listeners e scheduler. Alternativa considerada: schema-per-tenant (rejeitada
por overhead de migration e connection pooling).

Status: accepted (revisado em out/2026: **o RLS descrito na versão original nunca esteve ativo**)

## O que mudou

A decisão original era Row-Level Security no PostgreSQL ativado por `SET app.current_tenant = ?` injetado a cada
requisição. Isso **nunca funcionou**, por três motivos independentes, todos verificados contra o banco:

1. **Sintaxe:** `SET` não aceita parâmetro (`syntax error at or near "$1"`).
2. **Escopo:** `SET LOCAL` só vale dentro de uma transação, e o handle não abre nenhuma.
3. **Papel do banco:** a aplicação conecta como `postgres` (superusuário, `BYPASSRLS`), que ignora RLS mesmo com
   `FORCE ROW LEVEL SECURITY`. Sem `set_config`, o superusuário enxergava todas as linhas das tabelas com política.

O `IsAuth` deixou de executar o comando (economiza 1 round-trip por requisição protegida e o erro de log).

## Consequências

- **A defesa é o filtro `"tenantId"` explícito.** Toda query nova precisa dele; `auth.GetScoped` o aplica por padrão.
  `internal/middleware/auth_isolation_test.go` trava o caminho completo token → `IsAuth` → `GetScoped` → consulta,
  incluindo os ramos de atendente (`Tickets`, `Contacts`, padrão).
- As 10 políticas de RLS (`applyRLS`) **continuam criadas** no banco e são inofensivas com o usuário atual; não
  protegem nada e não devem ser citadas como proteção.
- Scripts administrativos e workers não dependem de RLS: continuam com `WHERE "tenantId"` manual.

## Se um dia se quiser RLS de verdade (change própria, não implementado)

Exige, nesta ordem: papel de aplicação `NOSUPERUSER NOBYPASSRLS`; transação por requisição com
`set_config('app.current_tenant', $1, true)`; política com exceção explícita para caminhos sem tenant (login, workers,
event listener, scheduler, webhooks, migrations), porque `current_setting(..., true)` devolve `NULL` onde não houve
`SET` e a política esconderia todas as linhas desses caminhos; revisão dos pontos que já abrem transação; e testes de
vazamento entre empresas por rota. É risco alto (mexe em autenticação) e fica fora desta decisão.

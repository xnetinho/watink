# Proposal

## Why

Toda requisição autenticada gera este erro no log do `business`:

```
auth.go:73 ERROR: syntax error at or near "$1" (SQLSTATE 42601)
SET LOCAL app.current_tenant = '2e9e9bf1-...'
```

Isso acontece em 100% das rotas protegidas, a cada chamada, e o comando **nunca funcionou**. Três defeitos
independentes, os três verificados contra o Postgres real:

1. **Sintaxe:** `SET` não aceita parâmetro (`syntax error at or near "$1"`, reproduzido no `psql`).
2. **Escopo:** `SET LOCAL` só vale dentro de uma transação, e o handle (`db.Session(...)`) não abre nenhuma.
3. **Papel do banco:** `DB_USER=postgres` é superusuário com `BYPASSRLS`, que ignora RLS mesmo com `FORCE ROW LEVEL
   SECURITY`. Medido: sem `set_config`, o superusuário enxerga todas as linhas, e um papel comum enxerga zero.

Resultado: 1 round-trip desperdiçado ao banco por requisição, erro falso no log, e uma falsa impressão de
segurança: o ADR 0001 ainda descreve o RLS como o mecanismo de isolamento. O isolamento real é o
`WHERE "tenantId"` explícito (232 chamadas a `auth.GetScoped`, mais as consultas de worker).

## What Changes

Decisão do dono: **Opção A, remover e declarar o RLS inerte.**

- Remove o `tx.Exec("SET LOCAL ...")` e o `db.Session` de `middleware/auth.go`; o handle do contexto passa a ser o
  `db` injetado.
- Teste que prova que o `IsAuth` não executa SQL e teste de regressão de isolamento entre empresas (garante que o
  isolamento nunca dependeu do `SET`).
- Atualiza o ADR 0001 (parcialmente superado), `CONTEXT.md` e o plano de QA que citam o comando.

As políticas de RLS continuam criadas no banco: são inofensivas e preservam a opção de ligar o RLS no futuro.

### Fora do escopo

- **Ligar o RLS de verdade** (transação por requisição, papel sem superusuário, exceção para workers/listeners/login/
  webhooks): risco alto, mexe em autenticação e em 23 pontos que já abrem transação. Só como change própria, com ADR
  e testes de vazamento entre empresas.
- O logger do GORM: change `fix-gorm-log-noise`.

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| Algum código depende do `app.current_tenant` | Nenhum encontrado | `Grep current_tenant`: só o `auth.go:73` e a criação das políticas |
| Ficar sem defesa em profundidade | Já era o caso | O RLS nunca filtrou nada (item 3); o teste de isolamento pega regressão do `WHERE "tenantId"` |
| `c.Set("db", ...)` muda de handle | Baixo | Mesmo `*gorm.DB`; o `db.Session(&gorm.Session{})` sem opções só clonava o handle |

# Tasks

> Fonte: `proposal.md` e `design.md`. Branch `fix/auth-rls-set-local`, PR contra `develop`. Todo teste novo é provado
> por mutação.

## 1. Remover o comando inoperante

- [x] 1.1 Teste que falha hoje: `IsAuth` não executa nenhuma instrução SQL (conta as consultas do handle)
- [x] 1.2 Remover o `tx.Exec("SET LOCAL ...")` e o `db.Session` de `middleware/auth.go`; ajustar o comentário da
      validação do UUID (a validação fica)
- [x] 1.3 Mutação: recolocar o `Exec` faz o teste do 1.1 falhar
- [x] 1.4 `Grep current_tenant`: confirmar que nada além da criação das políticas depende do setting

## 2. Isolamento entre empresas

- [x] 2.1 Teste de regressão contra Postgres real: duas empresas com mensagens/tickets; o handle de `auth.GetScoped`
      de A não devolve nada de B, e o de B não devolve nada de A (inclui os ramos de atendente `Tickets` e padrão)
- [x] 2.2 Mutação: remover o filtro de empresa de uma leitura faz o teste falhar

## 3. Documentação

- [x] 3.1 ADR 0001: status "parcialmente superado": RLS inerte, defesa é o `WHERE "tenantId"` manual
- [x] 3.2 `CONTEXT.md` (tirar "correção em andamento") e `docs/qa/plano-qa-pos-refatoracao-acessos-onboarding.md:279`

## 4. Verificação e entrega

- [x] 4.1 Ao vivo: subir o `business` contra Postgres real, autenticar, chamar uma rota protegida e conferir que o log
      não tem `syntax error at or near "$1"`
- [x] 4.2 `go build ./...`, `go vet`, `go test ./internal/middleware/ ./internal/controllers/` (controllers em segundo plano)
- [x] 4.3 `openspec validate fix-auth-rls-set-local --strict`; commit `fix(auth): ...`; PR contra `develop`

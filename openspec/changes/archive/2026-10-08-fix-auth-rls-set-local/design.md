# Design

## O que o código faz (`business/internal/middleware/auth.go:72-74`)

```go
tx := db.Session(&gorm.Session{})
tx.Exec("SET LOCAL app.current_tenant = ?", tenantID)
c.Set("db", tx)
```

## Por que não basta consertar a sintaxe

Trocar por `SELECT set_config('app.current_tenant', $1, true)` faz o erro sumir e o comando continua sem efeito:

- O terceiro argumento `true` é "local à transação"; sem transação vale só para aquela instrução, e a consulta
  seguinte pega outra conexão do pool.
- O usuário `postgres` ignora RLS. Provado: `rolsuper=true, rolbypassrls=true`, `FORCE RLS=true` em `Messages`, e
  mesmo assim 2 de 2 linhas visíveis sem nenhum `set_config`.
- Ligar o RLS parcialmente quebra tudo que roda sem tenant: login, workers, event listener, scheduler, webhooks e
  migrations. A política é `tenantId = current_setting('app.current_tenant', true)`, e `current_setting(..., true)`
  devolve `NULL` onde não houve `SET`, então a política esconderia todas as linhas desses caminhos.

## Decisão (Opção A)

Remover o `Exec` e o `Session`. `c.Set("db", db)`. O isolamento continua sendo o `WHERE "tenantId"` explícito; a
documentação passa a dizer isso.

O comentário do código ("Validate UUID to prevent SQL injection before string concatenation in SET LOCAL") perde o
motivo. A validação do UUID **fica**: o `tenantId` do token precisa ser um UUID válido para as consultas.

## Opção B (change futura, só se o dono quiser)

Exigiria, nesta ordem: ADR; papel de aplicação `NOSUPERUSER NOBYPASSRLS`; transação por requisição com
`set_config(..., true)`; política com exceção explícita para caminhos sem tenant; revisão dos 23 pontos que já abrem
transação; testes de vazamento entre empresas por rota. Não faz parte desta change.

## Verificação

- Teste que conta as instruções SQL que o `IsAuth` executa (hoje 1, depois 0).
- Teste de isolamento contra Postgres real: duas empresas, o handle de `GetScoped` de uma não devolve linhas da outra.
- Mutação: recolocar o `Exec` faz o primeiro falhar; remover o `WHERE "tenantId"` de uma rota de leitura faz o segundo
  falhar.
- Ao vivo: subir o `business` e confirmar que o log não tem mais `syntax error at or near "$1"`.

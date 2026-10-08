# Tasks

> Fonte: `proposal.md` e `design.md`. Branch `fix/gorm-log-noise`, PR contra `develop`. Todo teste novo é provado
> por mutação.

## 1. Logger do GORM

- [x] 1.1 Testes que falham hoje: `gormLogLevel` (válido, maiúsculas, inválido, ausente → `warn`); saída do logger
      contra Postgres real: `First` sem linha **não** imprime, e em `info` o valor do parâmetro **não** aparece
- [x] 1.2 `newGormLogger()` e `gormLogLevel()` em `internal/database` (arquivo próprio, `database.go` já é grande);
      `Connect` passa a usá-lo
- [x] 1.3 Mutação: tirar cada um de `IgnoreRecordNotFoundError`, `ParameterizedQueries` e a leitura de
      `DB_LOG_LEVEL` faz um teste falhar
- [x] 1.4 Documentar `DB_LOG_LEVEL` em `.env.production.example` e `docs/dev/commands.md`

## 2. Verificação e entrega

- [x] 2.1 Ao vivo: subir o `business` contra Postgres real, provocar um `First` sem linha e uma gravação com texto, e
      comparar o log antes e depois
- [x] 2.2 `go build ./...`, `go vet` e `go test ./internal/database/ ./internal/saasclient/ ./internal/services/`
- [x] 2.3 `openspec validate fix-gorm-log-noise --strict`; commit `fix(database): ...`; PR contra `develop`

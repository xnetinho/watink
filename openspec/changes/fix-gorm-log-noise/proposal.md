# Proposal

## Why

O log do `business` em homolog (out/2026) tem dois problemas que vêm da mesma linha de configuração, o logger do
GORM em `database.Connect`:

1. **Ruído de erro falso.** `saas_contract_service.go:59 record not found` a cada 30 s (e `skeleton.go:297` a cada
   mensagem). É o estado normal de uma instância sem Modo SaaS e de um ticket sem fluxo ativo, e o código já trata
   `gorm.ErrRecordNotFound`. O logger padrão do GORM imprime esse caso como erro porque
   `IgnoreRecordNotFoundError` não está ligado, e o erro de verdade se perde no meio.
2. **Vazamento de dado pessoal.** O logger roda em `LogMode(logger.Info)` em produção e imprime **toda** consulta
   SQL **com os valores**: texto das mensagens de clientes, IDs de tenant, contatos. Isso vai para o agregador de
   logs do operador. Um exemplo do log real: `INSERT INTO "Messages" (... 'Sim' ...)`.

## What Changes

- `database.Connect` passa a montar o logger do GORM com `IgnoreRecordNotFoundError: true`,
  `ParameterizedQueries: true`, `SlowThreshold: 200ms` e nível vindo de `DB_LOG_LEVEL`
  (`silent|error|warn|info`, padrão `warn`).
- Documenta `DB_LOG_LEVEL` em `.env.production.example` e `docs/dev/commands.md`.

### Fora do escopo

- Trocar `First` por `Take` nos chamadores (a causa é única; o logger resolve todos de uma vez).
- Ocultar os valores de outros logs (`log.Printf` espalhados): só o logger do GORM.
- O `SET LOCAL app.current_tenant`: change `fix-auth-rls-set-local`.

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| Alguém depende de ver o SQL no log para depurar | Baixo | `DB_LOG_LEVEL=info` mostra as consultas, sem os valores |
| `warn` esconde uma consulta que hoje se vê | Baixo | Erro de SQL e consulta lenta (>200 ms) continuam impressos |

# Design

## Causa

`business/internal/database/database.go:29`:

```go
Logger: logger.Default.LogMode(logger.Info),
```

`logger.Default` trata `ErrRecordNotFound` como erro (`IgnoreRecordNotFoundError` é `false`), e `LogMode(Info)` imprime
todo SQL com os parâmetros interpolados.

## Decisão

Um único ponto de correção:

```go
func newGormLogger() logger.Interface {
    return logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
        SlowThreshold:             200 * time.Millisecond,
        LogLevel:                  gormLogLevel(os.Getenv("DB_LOG_LEVEL")),
        IgnoreRecordNotFoundError: true,
        ParameterizedQueries:      true,
    })
}
```

- **`IgnoreRecordNotFoundError`**: corrige `saas_contract_service.go:59`, `skeleton.go:297` e todos os iguais sem
  tocar em cada chamador. O chamador já decide o que fazer com o "não achei".
- **`ParameterizedQueries`**: mesmo em `info`, o valor vira `$1`. Corta o vazamento.
- **`LogLevel`**: `warn` por padrão. `gormLogLevel` aceita `silent|error|warn|info` (sem diferenciar maiúsculas) e
  cai em `warn` para ausente ou inválido: um erro de digitação em produção não pode ligar o `info`.
- **Sem nova dependência**: tudo vem de `gorm.io/gorm/logger`.

## Alternativas descartadas

- **`Take` em vez de `First` em cada chamador**: dezenas de pontos, e o próximo `First` novo reintroduz o ruído.
- **Manter `info` e só ocultar os valores**: o log continuaria enorme e barulhento.
- **`silent`**: esconderia erros reais de SQL e consultas lentas.

## Verificação

Teste de unidade do mapeamento do nível e teste que captura a saída do logger contra Postgres real (`First` sem
linha não imprime; em `info` o texto do parâmetro não aparece). Mutação: desligar cada um dos três ajustes faz um teste falhar.

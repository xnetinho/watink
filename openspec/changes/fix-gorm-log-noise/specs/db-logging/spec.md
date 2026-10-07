# Spec Delta

## Purpose

Mantém o log do servidor útil e sem dado pessoal: só erros reais e consultas lentas por padrão, sem os valores dos
parâmetros das consultas.

## ADDED Requirements

### Requirement: Ausência de linha não é erro de log
O sistema SHALL NOT imprimir no log uma consulta que não encontrou registro (`gorm.ErrRecordNotFound`).

#### Scenario: Consulta sem resultado
- **WHEN** um `First` não encontra linha e o chamador trata `gorm.ErrRecordNotFound`
- **THEN** nada é impresso no log

### Requirement: Valores de parâmetros não vão para o log
O sistema SHALL NOT imprimir o valor dos parâmetros das consultas SQL, em nenhum nível de log.

#### Scenario: Texto de mensagem de cliente
- **WHEN** `DB_LOG_LEVEL=info` e uma consulta recebe o texto de uma mensagem de cliente como parâmetro
- **THEN** a consulta aparece no log com o marcador do parâmetro e sem o texto

### Requirement: Nível de log configurável com padrão seguro
O sistema SHALL ler o nível do log do banco de `DB_LOG_LEVEL` (`silent`, `error`, `warn`, `info`) e SHALL usar `warn`
quando o valor estiver ausente ou inválido.

#### Scenario: Valor inválido
- **WHEN** `DB_LOG_LEVEL` tem um valor desconhecido
- **THEN** o nível é `warn`

#### Scenario: Consulta lenta continua visível
- **WHEN** uma consulta leva mais de 200 ms em nível `warn`
- **THEN** ela é impressa no log

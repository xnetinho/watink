# Spec Delta

## Purpose

A autenticação não executa SQL, e o isolamento entre empresas é o filtro `"tenantId"` explícito em cada consulta.

## ADDED Requirements

### Requirement: Autenticação não executa SQL
O sistema SHALL autenticar a requisição sem executar nenhuma instrução SQL no banco.

#### Scenario: Rota protegida sem erro de SQL
- **WHEN** uma requisição autenticada chega a uma rota protegida
- **THEN** o middleware de autenticação não consulta o banco e o log não contém `syntax error at or near "$1"`

### Requirement: Isolamento entre empresas por filtro explícito
O sistema SHALL isolar os dados de cada empresa por filtro `"tenantId"` explícito em cada consulta, e SHALL NOT
depender de política de RLS para esse isolamento.

#### Scenario: Uma empresa não enxerga a outra
- **WHEN** o handle de contexto da empresa A consulta mensagens
- **THEN** nenhuma linha da empresa B é devolvida

#### Scenario: Token com empresa inválida
- **WHEN** o token tem um `tenantId` que não é um UUID válido
- **THEN** a requisição é recusada com 401

# Spec Delta

## Purpose

Mantém o repositório livre de código e configuração que descrevem coisas que não existem mais, sem mudar comportamento.

## ADDED Requirements

### Requirement: Remoção segura de código morto
O sistema SHALL remover código somente quando o grafo de imports ou o `deadcode` indicar ausência de uso e uma busca
manual confirmar, e SHALL manter código usado só por testes, código vendorizado e código de feature em andamento.

#### Scenario: Remoção não quebra o build
- **WHEN** um lote de remoção é aplicado
- **THEN** o build, o typecheck, o lint e os testes dos módulos tocados passam

#### Scenario: Falso positivo de ferramenta é preservado
- **WHEN** uma dependência é usada por um script fora de `src/` ou carregada por string em configuração
- **THEN** ela não é removida

### Requirement: Configuração aponta só para o que existe
O sistema SHALL manter `dependabot`, `codeql`, workflows, scripts e `.gitignore` referenciando somente pastas e
arquivos existentes, e SHALL cobrir todos os módulos com dependências no `dependabot`.

#### Scenario: Módulo sem atualização automática
- **WHEN** um módulo com `go.mod` ou `package.json` não tem entrada no `dependabot`
- **THEN** a entrada é adicionada

### Requirement: Documentação não descreve serviços removidos
O sistema SHALL NOT listar como existente, na documentação operacional (`CLAUDE.md`), um serviço ou pasta que não
existe no repositório.

#### Scenario: Tabela de serviços
- **WHEN** o `CLAUDE.md` lista os serviços do projeto
- **THEN** cada pasta citada existe

# Spec Delta

## Purpose

O build de produção do frontend não publica o código-fonte original.

## ADDED Requirements

### Requirement: Build de produção sem mapas de fonte
O sistema SHALL NOT publicar arquivos de mapa de fonte (`.map`) nem referências `sourceMappingURL` no build de
produção, por padrão.

#### Scenario: Build padrão
- **WHEN** o frontend é compilado sem `VITE_SOURCEMAP`
- **THEN** o diretório de saída não contém nenhum arquivo `.map` e nenhum `.js` referencia um mapa

#### Scenario: Visitante abre o F12
- **WHEN** um visitante abre as ferramentas do navegador numa instância em produção
- **THEN** o código TypeScript original não está disponível para leitura

### Requirement: Mapas apenas por pedido explícito
O sistema SHALL gerar mapas de fonte somente quando `VITE_SOURCEMAP` for exatamente `true` (sem diferenciar
maiúsculas nem espaços), e SHALL manter desligado para qualquer outro valor.

#### Scenario: Depuração pontual
- **WHEN** o build roda com `VITE_SOURCEMAP=true`
- **THEN** os mapas de fonte são gerados

#### Scenario: Valor com erro de digitação
- **WHEN** o build roda com `VITE_SOURCEMAP=ture`
- **THEN** nenhum mapa é gerado

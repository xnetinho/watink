# Proposal

## Why

A abstração de canais (ADR 0033) foi declarada pronta, mas o miolo do sistema continua acoplado ao WhatsApp: responder
um ticket do Telegram devolve 404, o FlowBuilder só fala com `"whatsapp"`, o adaptador WhatsApp é uma casca, existem duas
tabelas de conexão e `whatsappId` em 9 tabelas, o contrato com o engine é escrito à mão nos dois lados (dois eventos
divergem e um nunca é emitido), o engine perde eventos quando o RabbitMQ cai, 80 das 214 rotas protegidas não exigem
permissão e 62 de 82 controllers falam direto com o GORM. Sem instância em produção, este é o momento de corrigir a
estrutura em vez de remendá-la.

## What Changes

- **Uma só noção de conexão:** toda conexão é `ChannelConnection`; o `sessionId` do engine é o `connectionId`.
  `Whatsapps` deixa de existir como cabeça (campos específicos viram detalhe 1:1), `whatsappId` some das 9 tabelas.
- **Identidade de mensagem composta** `(tenantId, connectionId, externalId)` com chave primária interna.
- **`MessageDispatchService`:** único caminho de saída (controllers, flows, assistentes, campanhas, respostas rápidas,
  plugins), com `ReplyPolicy` e validação de `Capabilities`.
- **Adaptador WhatsApp real** e remoção dos ramos `EngineType != "whatsmeow"`; **registro único de canais**.
- **Contrato engine↔business** em módulo compartilhado (`contracts/wbot`) com construtor único de routing key, teste de
  contrato bidirecional e handshake `engine.ready` no lugar do SQL de reinício.
- **Engine resiliente:** canais AMQP separados, confirmação de publicação, saída durável de eventos, `Nack` com DLQ,
  consumidor que se refaz após reconexão.
- **Autorização:** `RequirePermission` em toda mutação, com teste de tabela de rotas que reprova rota sem permissão.
- **Business em camadas:** controllers finos, repositórios por empresa, barramento de eventos durável, `recover` em
  toda goroutine e consumidor.
- **Chamadas e vídeo como `Capabilities`**; vídeo permanece **experimental** atrás de flag até validação ao vivo.
- **Frontend agnóstico:** conexões únicas, sem JID no cliente, ticket com `channel` e `capabilities`.
- **Operação:** métricas Prometheus, readiness real, erro de canal visível na UI, CI com `-race` e testes do engine.
- **Canal `fake` de teste** como prova objetiva de que o core é agnóstico.
- **BREAKING:** o banco é recriado do zero; todo WhatsApp precisa ser pareado de novo (QR) quando o armazenamento do
  `whatsmeow` for refeito; rotas `/whatsapp/*` e `whatsappId` somem da API.

## Capabilities

### New Capabilities
- `channel-messaging`: conexão única, identidade de mensagem, envio e recebimento unificados por capacidades de canal.
- `engine-contract`: contrato versionado e resiliente entre o engine do WhatsApp e o business.
- `api-authorization`: toda rota protegida declara a permissão exigida, verificado automaticamente.
- `platform-operations`: gates de CI, métricas, readiness e visibilidade de falhas.

### Modified Capabilities

## Impact

- `business/` (controllers, services, flow, plugins, models, database, routes), `engine-go/` (rabbitmq, whatsapp, calls),
  novo módulo `contracts/` com `go.work`, `frontend/src` (conexões, tickets, composer), `.github/workflows`, Dockerfiles,
  `docs/adr`, `openspec/changes/add-channel-*` (reconciliar status das tarefas).
- Depende de `enforce-tenant-rls` apenas para o esquema greenfield; as duas mudanças são independentes na ordem de entrega
  exceto onde indicado nas tarefas.

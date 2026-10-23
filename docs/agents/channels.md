# Canais Agnósticos de Comunicação — Contexto para Agentes

> **Status:** em implementação (ADR 0033 / change `add-channel-abstraction`).

## Responsabilidade
Permitir que o Watink se comunique com múltiplos canais de mensageria (**WhatsApp, Telegram, Mercado Livre, OLX, Instagram, TikTok**), desacoplando o núcleo de negócio de particularidades exclusivas do WhatsApp. A fundação estabelece modelo de dados unificado, interface agnóstica de canal, capacidades em dado, regras de resposta e webhooks genéricos, garantindo paridade e regressão zero no WhatsApp.

## Arquitetura
```
[Provedor Externo] (Webhook HTTP / Polling)
        │
        ▼
POST /webhooks/channels/:kind/:connectionId  (HMAC / segredo por conexão)
        │
        ▼
   ChannelInbound.Receive (dedup no Redis por [connection, externalMessageId])
        │
        ▼
   ReceiveMessageUseCase (criação/atualização de Ticket via conversationKey, ContactIdentity)
        │
        ▼
  Mensageria / SSE / UI (Catálogo de conexões, Composer guiado por Capabilities)
```

- **business/internal/domain/channel**: Interface `Channel`, `Capabilities`, `ReplyPolicy`, `InboundMessage`, `OutboundMessage`.
- **business/internal/channels**: Adaptadores de canal (`whatsapp`, e futuros `telegram`, `mercadolivre`, etc.).
- **business/models**: `ContactIdentity`, `ChannelConnection` (generalizando `Whatsapps`), `Ticket` (`channelConnectionId`, `conversationKey`).
- **frontend/src**: Catálogo de canais em `/connections`, badges nos tickets, composer adaptativo.

## Invariantes e Regras de Ouro
1. **Expandir e Contrair (Regressão Zero)**:
   - Toda alteração estrutural no banco é aditiva: novas tabelas e colunas nullable ao lado das legadas, com backfill transparente e leitura dupla.
   - O canal WhatsApp mantém seu comportamento funcional inalterado. A tabela `Whatsapps` e colunas `Ticket.whatsappId` continuam funcionando até contração final.
2. **Identidade do Contato (`ContactIdentity`)**:
   - `Contact` representa a pessoa/entidade única de atendimento no CRM.
   - Um `Contact` pode possuir N `ContactIdentity` (`tenantId`, `channel`, `externalId`).
   - Vínculo manual ou explícito entre contatos de canais diferentes; nunca unificação automática por suposição.
3. **Chave de Conversa (`conversationKey`)**:
   - Em canais centrados em pessoa (WhatsApp, Telegram, Instagram): `conversationKey` mapeia o identificador do contato.
   - Em canais centrados em transação/objeto (Mercado Livre, OLX): `conversationKey` representa o pedido (`pack_id`/`order_id`) ou o anúncio (`ad_id`), permitindo múltiplos tickets independentes com o mesmo comprador.
4. **Regras de Resposta no Servidor (`ReplyPolicy`)**:
   - Restrições de tempo ou iniciativa de resposta (janela de 24h no Instagram, 48h úteis no Mercado Livre) são avaliadas pelo backend (`CanReply`) e expostas ao frontend.
   - Violações barram envio com HTTP 409 e motivo claro (`reply_window_expired`, `initiator_not_allowed`), não deixando falhar silenciosamente no webhook externo.
5. **Segurança nos Webhooks**:
   - Rotas de webhook validam assinatura criptográfica / secret específico por conexão.
   - Falhas de assinatura recusam silenciosamente ou retornam erro sem revelar existência de tenants/conexões.

## O que NÃO fazer
- Não assumir que todo canal tem número de telefone (JID ou telefone só são garantidos no WhatsApp e OLX).
- Não criar colunas específicas de um novo canal direto na tabela `Tickets` ou `Whatsapps`.
- Não quebrar o fluxo de eventos AMQP do WhatsApp (`wbot.<tenant>.<session>.*`).
- Não forçar abstração prematura no `engine-go`: canais baseados em HTTP/Webhook rodam nativamente dentro do `business`. O `engine-go` permanece dedicado às particularidades de socket/VoIP do WhatsApp.

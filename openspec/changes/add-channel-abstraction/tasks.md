# Tasks

> Fonte: `proposal.md` e `design.md`. **Regra de ouro: expandir e contrair.** Cada fase é aditiva, mantém o WhatsApp no
> caminho atual e tem teste de regressão de que o comportamento dele não mudou. Um PR por item numerado (ou menor),
> verificado localmente (`docs/dev/local-verification.md`) **antes** do push. As 5 integrações dependem das **fases 0 a 4**.
> Convenções: `auth.GetScoped`, `Session(NewDB:true)`, testes contra Postgres real, lint v2.12.2 local, mutação nos testes.

## 0. Decisões que bloqueiam (HITL)

- [ ] 0.1 Aprovar o modelo `Channel` + `Capabilities` (design D1) e a `conversationKey` (D4)
- [ ] 0.2 Aprovar a ordem das integrações (proposta: Telegram → Mercado Livre → Instagram → OLX → TikTok) e quais pedidos
      de aprovação às plataformas devem **começar já** (Instagram App Review, TikTok DSPR, OLX registro)
- [ ] 0.3 Decidir se o `Whatsapp` migra para `ChannelConnection` de vez ou se as duas tabelas coexistem (D5/D7)
- [ ] 0.4 Decidir o destino da fila única de eventos (dívida do CLAUDE.md) antes de ligar canais de volume

## 1. Glossário e ADR

- [ ] 1.1 ADR 0033 (canais agnósticos) e entradas em `CONTEXT.md`: `Channel`, `ChannelConnection`, `ContactIdentity`,
      `conversationKey`, `Capabilities`. Resolver o conflito com o termo atual `Whatsapp` ("Avoid: channel")
- [ ] 1.2 `docs/agents/channels.md` (módulo novo) e seção em `CLAUDE.md`

## 2. Fundação de dados (aditiva)

- [ ] 2.1 `ContactIdentity` + migration + **backfill** de `Contact.number`/`lid` → identidades `whatsapp`; teste de que
      todo contato existente ganha identidade e que o backfill é idempotente
- [ ] 2.2 `ChannelConnection` (kind, nome único **por tenant**, `credentialsEnc` cifrado, `config`, status) ao lado de
      `Whatsapps`; `channel_connection_queues`
- [ ] 2.3 `Ticket.channelConnectionId` e `Ticket.conversationKey` (nullable, backfill = `whatsappId` e `contactId`)
- [ ] 2.4 Índice único `(tenantId, channelConnectionId, conversationKey)` para ticket aberto, **sem** remover o atual
- [ ] 2.5 Teste de regressão: `FindOpenByContact` do WhatsApp devolve o mesmo ticket que antes (paridade provada)

## 3. Interface, entrada e regra de resposta

- [ ] 3.1 `domain.Channel`, `Capabilities`, `InboundMessage`, `OutboundMessage` e o registro por `Kind`
- [ ] 3.2 `ChannelInbound.Receive`: contrato único de entrada com dedup `(connection, externalMessageId)` no Redis;
      `ReceiveMessageUseCase` passa a consumi-lo (o `MessagePayload` do WhatsApp vira um conversor)
- [ ] 3.3 `CanReply` + `replyPolicy` em `GET /tickets/:id` e recusa `409 {code}` em `POST /messages`
- [ ] 3.4 Webhook genérico `POST /webhooks/channels/:kind/:connectionId` com segredo por conexão (HMAC/header), recusa
      assinatura inválida sem revelar se a conexão existe
- [ ] 3.5 Suíte de **testes de contrato compartilhada** que todo adaptador roda (idempotência, dedup, janela, limite de
      texto, credencial inválida)
- [ ] 3.6 SSE: o evento `whatsappSession` ganha equivalente neutro `channelConnection` (o antigo continua emitido)

## 4. WhatsApp como o primeiro adaptador (paridade, sem mudar comportamento)

- [ ] 4.1 Adaptador `whatsapp` que delega ao `WhatsAppEngine`/`enginego`/`izapia` existentes
- [ ] 4.2 Os 5 pontos com `if EngineType != "whatsmeow"` passam pelo registro de canais
- [ ] 4.3 Uma função única de endereçamento (hoje há ao menos 6 construtores de JID espalhados)
- [ ] 4.4 Teste de regressão ponta a ponta do WhatsApp (receber, enviar, ack, citação, mídia) contra Postgres e
      RabbitMQ reais **antes e depois**: mesmos resultados

## 5. Frontend: conexões, ticket e composer

- [ ] 5.1 Seletor de canal e catálogo em `/connections` (cartões com selo de disponibilidade)
- [ ] 5.2 Assistentes de conexão por `AuthKind` (`token`, `oauth`, `oauth_review`; o `qr` já existe)
- [ ] 5.3 Badge de canal (ícone + token de cor) na lista e no cabeçalho do ticket; assets SVG em `src/assets/channels/`
- [ ] 5.4 Composer guiado por `Capabilities`/`replyPolicy` (contador, mídia, motivo do bloqueio, ação alternativa)
- [ ] 5.5 Contexto da conversa (pedido no ML, anúncio na OLX) no cabeçalho do ticket
- [ ] 5.6 Esconder recursos exclusivos de WhatsApp (chamadas, grupos, enquetes, botões) quando o canal não os declara
- [ ] 5.7 Filtro por canal na lista de tickets e no Dashboard; traduções pt/en/es; testes de componente e de acessibilidade

## 6. Contração (depois das 5 integrações, change própria)

- [ ] 6.1 Remover colunas e índices herdados só quando a leitura nova for a única; uma change própria, com ADR

## 7. Fechamento

- [ ] 7.1 `/qa-analyst` sobre a DAG; `openspec validate add-channel-abstraction --strict`

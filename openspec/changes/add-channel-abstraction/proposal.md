# Proposal

## Why

O Watink nasceu como plataforma de atendimento no **WhatsApp** e o código assume isso em toda parte: 318 arquivos Go e
141 TS/TSX citam "whatsapp"; o contato é identificado por `number`/`lid` (vocabulário de JID); a conexão é a tabela
`Whatsapps`; o engine fala `wbot.<tenant>.<session>.<cmd>`. O produto agora precisa receber e enviar mensagens por
**Telegram, Mercado Livre, OLX, Instagram e TikTok**, e cada um traria um acoplamento novo se o sistema continuasse
WhatsApp-cêntrico.

O precedente mais próximo, o **izapia** (segundo "engine" de WhatsApp), mostrou o custo de encaixar um segundo canal
sem abstração: colunas próprias na tabela `Whatsapps`, um webhook que traduz o payload para o vocabulário de JID
(`@g.us`), e um `if EngineType != "whatsmeow"` repetido em 5 pontos de envio. Funciona porque o izapia *continua sendo
WhatsApp*. Um canal sem JID (Telegram `chat_id`, Instagram IGSID, ML `pack_id`) **não se encaixa nesse caminho**.

Esta change planeja a **fundação comum**: uma interface de canal, uma identidade de contato por canal e uma UX de
conexões que não presuma WhatsApp. As cinco integrações são changes próprias que dependem desta.

## What Changes

- **Interface de canal** (`domain.Channel`), com capacidades declaradas (texto, mídia, grupos, janela de resposta,
  iniciar conversa...) e um mapa de implementações por tipo, no lugar do `if EngineType` espalhado.
- **Identidade de contato por canal**: `ContactIdentity(tenant, channel, externalId)` em tabela própria. `Contact`
  continua sendo a pessoa; ganha N identidades. Remove a dependência de `number`/`lid` como chave.
- **Conexão neutra**: generalização de `Whatsapp` para `ChannelConnection` (`kind`, `credentials` cifradas, `status`),
  mantendo `Whatsapp` como o caso `kind=whatsapp` durante a migração.
- **Entrada unificada**: `InboundMessage` neutro e um único ponto (`ChannelInbound.Receive`) que substitui o
  `MessagePayload` de WhatsApp como contrato de entrada, para webhook e polling de qualquer canal.
- **Regras de resposta por canal** (janela de 24 h do Instagram, "vendedor não inicia" do Mercado Livre, limites de
  caracteres) como dado da capacidade, aplicadas no backend, com o motivo devolvido ao frontend.
- **UX do frontend**: Conexões como catálogo de canais, assistente de conexão por tipo, ícone/badge de canal nos
  tickets, composer que respeita as capacidades, filtro por canal.
- **ADR 0033** registrando as decisões e o glossário (`Channel`, `ChannelConnection`, `ContactIdentity`).

### Fora do escopo

- Qualquer integração concreta (cada uma é uma change própria: Telegram, Mercado Livre, OLX, Instagram, TikTok).
- Reescrever o `engine-go` ou o WhatsApp: ele continua como o canal `whatsapp` e **não muda de comportamento**.
- Mensagens de saída em massa/campanhas para os novos canais (cada plataforma tem regra própria; ver changes).
- Unificar contatos de canais diferentes por heurística (mesma pessoa no Telegram e no WhatsApp): o vínculo é
  **manual**, como o ADR 0023 já faz para Contact↔Client.
- Grupos fora do WhatsApp e do Telegram.

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| Migrar `Whatsapps`/`Ticket.whatsappId` quebra o que funciona | **Alto** | Expandir-e-contrair: tabelas e colunas novas ao lado das antigas, backfill, leitura dupla, só depois remover. WhatsApp fica no mesmo caminho até o fim |
| Mudar a identidade do contato (`number`/`lid`) corrompe contatos existentes | **Alto** | `ContactIdentity` nasce por backfill a partir de `number`/`lid`; os índices únicos atuais ficam até a paridade ser provada |
| Abstração prematura que só serve a um canal | Médio | Desenhar com **dois** canais reais em mente (Telegram e Mercado Livre são opostos: aberto vs restrito); validar contra todos os cinco antes de fixar a interface |
| Fila única de eventos serializa todos os tenants | Médio | Já é dívida conhecida (CLAUDE.md); canais de marketplace de alto volume a agravam. Decidir filas por canal ou por shard antes de ligar volume |
| Escopo enorme numa change só | Alto | Fatiar em fases com PRs pequenos (ver `tasks.md`); cada fase entrega valor e não muda o WhatsApp |

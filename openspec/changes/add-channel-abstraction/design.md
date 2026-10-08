# Design

## Princípio

**O núcleo do Watink (Ticket, Contact, Message, Queue, Flow, RBAC, SSE) não sabe qual canal está usando.** Cada canal é um
adaptador que traduz entre o protocolo da plataforma e três contratos neutros: **receber**, **enviar** e **conectar**.
O WhatsApp passa a ser **o primeiro adaptador**, não o caso especial.

Isto continua o que o ADR 0014 começou (`flow.OutboundChannelAdapter`, só saída) e o que o izapia provou em pequeno
(`engines map[string]WhatsAppEngine`). Falta o lado da **entrada** e o **modelo de dados**.

## O teste que valida a abstração

Antes de fixar qualquer interface, cada dimensão foi conferida contra os seis canais (documentação oficial, out/2026):

| | WhatsApp | Telegram (bot) | Mercado Livre | OLX | Instagram | TikTok |
|---|---|---|---|---|---|---|
| Quem inicia a conversa | qualquer um | **só o usuário** | **só o comprador** | **só o comprador** | **só o usuário** | **só o usuário** |
| Janela para responder | nenhuma | nenhuma | 48 h úteis (depois bloqueia) | não documentada | **24 h** (+7 d humano) | 48 h (limite de msgs) |
| Identidade | JID / LID / fone | `chat_id` | `pack_id` (+ agente de IA) | `chatId` + `listId` | IGSID | `conversation_id` |
| Telefone | sim | **não** | a confirmar | sim | **não** | **não** |
| Conexão do cliente | QR / pairing | colar token | OAuth | OAuth + registro | OAuth + App Review | OAuth + revisão DSPR |
| Entrega de entrada | AMQP (engine) | webhook ou polling | webhook + feed 2 d | webhook | webhook | webhook |
| Texto máximo | ~65 k | 4096 | **350** | não documentado | 1000 bytes | não documentado |
| Mídia de saída | tudo | tudo | JPG/PNG/PDF/TXT 25 MB | só texto doc. | img/áudio/vídeo/PDF | texto + imagem |
| Grupos | sim | sim | não | não | não | não |
| "Digitando" | sim | sim | não | não | sim | sim |
| Ack de leitura | sim | não | `date_read` | não | `seen` | `read` |

Cinco achados **mudaram** o desenho original:

1. **"Quem inicia" e "janela" são regras de resposta por canal**, não detalhe de provider. Cinco dos seis canais
   restringem. Têm de ser **capacidades declaradas** e **aplicadas no backend** (e devolvidas à UI), senão o atendente
   digita e só descobre o erro depois do envio.
2. **A conversa nem sempre é uma pessoa.** No Mercado Livre é um **pedido** (`pack_id`); na OLX, um **anúncio**
   (`listId`). A mesma pessoa comprando duas vezes é duas conversas. Isso **conflita** com a regra atual
   `FindOpenByContact(contato, conexão)` ("um ticket aberto por contato e conexão"). Precisa de uma **chave de
   conversa** além do contato.
3. **Telefone não é identidade universal.** Só WhatsApp e OLX garantem. `Contact.number`/`lid` deixa de ser a chave.
4. **Grupos só existem em WhatsApp e Telegram.** Já é opcional no código atual (`GroupEngine` por type-assertion).
5. **A entrega de entrada varia** (AMQP, webhook, polling): precisa de **um único ponto de entrada** neutro.

## Decisões

### D1. Interface de canal pequena, capacidades declaradas

```go
// internal/domain/channel.go
type Channel interface {
    Kind() ChannelKind                              // "whatsapp" | "telegram" | "mercadolivre" | "olx" | "instagram" | "tiktok"
    Capabilities() Capabilities                     // o que o canal PODE fazer (dado, não código)
    Connect(ctx, ChannelConnection, ConnectInput) (ConnectResult, error)   // QR, token, OAuth...
    Disconnect(ctx, ChannelConnection) error
    Send(ctx, ChannelConnection, OutboundMessage) (SendResult, error)
    ParseInbound(ctx, ChannelConnection, RawEvent) ([]InboundMessage, error)   // webhook/polling → neutro
}

type Capabilities struct {
    InitiateConversation bool          // pode abrir conversa com quem nunca falou
    ReplyWindow          *time.Duration // nil = sem janela; Instagram 24h; TikTok 48h
    ReplyWindowExtension *ReplyWindowExtension // ex.: tag HUMAN_AGENT +7 dias (Instagram)
    MaxTextBytes         int
    Media                []MediaKind
    Groups               bool
    Typing, ReadReceipts, Reactions bool
    Auth                 AuthKind      // qr | token | oauth | oauth_review
    HasPhone             bool
}
```

Recursos que só alguns canais têm continuam **interfaces opcionais por type-assertion** (padrão já usado:
`RichMessageEngine`, `GroupEngine`, `PresenceEngine`). A interface base não cresce.

### D2. Regra de resposta no backend, motivo na UI

`CanReply(conversation, now) → (ok bool, reason ReplyBlockReason, until *time.Time)` é calculado do `Capabilities` e do
último inbound. Devolvido em `GET /tickets/:id` (`replyPolicy`) e reavaliado no `POST /messages`. A UI **desabilita o
composer com o motivo** ("a janela de 24 h do Instagram expirou há 3 h") e oferece o que for possível (ex.: tag de
atendente humano). O backend recusa com 409 + `code`, nunca confia no frontend.

### D3. Identidade de contato por canal (`ContactIdentity`)

```
ContactIdentity(id, tenantId, contactId, channelKind, externalId, displayName, avatarUrl, meta jsonb)
UNIQUE (tenantId, channelKind, externalId)
```

`Contact` continua sendo **a pessoa** (nome, cliente CRM, carteira) e ganha N identidades. O vínculo entre identidades
de canais diferentes é **manual** (mesmo princípio do ADR 0023). O WhatsApp faz **backfill** de `number`/`lid` →
identidades `whatsapp`; os índices únicos atuais ficam até a paridade ser provada.

### D4. Chave de conversa (`conversationKey`)

`Ticket` ganha `channelConnectionId` e `conversationKey` (texto). A regra de "ticket aberto" passa a ser
`(tenantId, channelConnectionId, conversationKey)`:

| Canal | `conversationKey` |
|---|---|
| WhatsApp, Telegram, Instagram, TikTok | o `externalId` do contato (1 por pessoa, como hoje) |
| **Mercado Livre** | `pack_id` (ou `order_id`): **1 ticket por pedido** |
| **OLX** | `listId`:`chatId` (1 ticket por anúncio) |

Assim a recompra no ML abre outro ticket, sem fundir pedidos diferentes, e o histórico do **contato** continua
agregado por `ContactIdentity`.

### D5. Conexão neutra (`ChannelConnection`)

Generalização de `Whatsapp`: `kind`, `name` (único **por tenant**, hoje é único global), `status`, `credentialsEnc`
(JSON cifrado com `cryptobox`, mesmo padrão do Proxy), `config jsonb`, `lastError`. Os campos específicos de WhatsApp
(`qrcode`, `wid`, `proxy*`, `izapia*`) ficam como `config` do kind `whatsapp` durante a migração. A associação com filas
(`whatsapp_queues`) vira `channel_connection_queues`; a visibilidade de ticket por fila (`pkg/auth/tenant.go`) passa a
usá-la.

### D6. Entrada única

`ChannelInbound.Receive(ctx, InboundMessage)` substitui o `MessagePayload` de WhatsApp como contrato do
`ReceiveMessageUseCase`. O consumidor AMQP do WhatsApp, o webhook do izapia e os webhooks dos novos canais viram
**produtores** do mesmo contrato. Dedup por `(channelConnectionId, externalMessageId)` no Redis (padrão `wbot:msg:`).
Os webhooks são autenticados por **segredo por conexão** (HMAC ou header), como o izapia já faz.

### D7. Expandir e contrair, nunca trocar

Cada passo é aditivo, com leitura dupla e backfill, e o WhatsApp **não muda de caminho** até a paridade estar provada:

1. criar tabelas e colunas novas ao lado das antigas;
2. escrever nos dois lados; 3. ler do novo com fallback ao antigo; 4. migrar o WhatsApp para o adaptador;
5. só depois remover o que ficou sem uso (change própria).

## UX do frontend

Princípio: **a interface escolhe o fluxo pelo canal; o resto do sistema não precisa saber.**

### Conexões como catálogo de canais
`/connections` lista as conexões **de todos os canais** com ícone, nome, status e um filtro por canal. O botão
"Nova conexão" abre um **seletor de canal** (cartões com ícone, descrição de uma linha, o que exige e um selo de
disponibilidade: *Disponível*, *Requer aprovação da plataforma*, *Beta*).

### Assistente de conexão por tipo (um componente por `AuthKind`)
| `Auth` | Passos |
|---|---|
| `qr` (WhatsApp) | QR / código de pareamento (hoje) |
| `token` (Telegram) | colar o token do @BotFather → o sistema valida (`getMe`) e registra o webhook → mostra o nome do bot |
| `oauth` (Mercado Livre) | botão "Conectar com Mercado Livre" → redireciona → retorna com a conta vinculada |
| `oauth_review` (Instagram, TikTok) | igual a `oauth`, com **aviso claro do que a plataforma exige** (App Review, verificação) e o estado "aguardando aprovação" |

Cada passo mostra **o que o canal vai poder e não poder fazer** (vindo de `Capabilities`), por exemplo "no Instagram você
só pode responder em até 24 h".

### Ticket e conversa
- **Badge de canal** (ícone + cor semântica por token do design system) na lista e no cabeçalho do ticket.
- **Composer guiado por capacidades:** limite de caracteres do canal (contador), tipos de mídia aceitos, anexo
  desabilitado quando o canal não suporta, e **o motivo** quando a resposta está bloqueada.
- **Contexto da conversa:** no ML o cabeçalho mostra o **pedido** (nº, item, status); na OLX o **anúncio**.
- Recursos exclusivos de WhatsApp (chamadas, grupos, enquetes, botões) só aparecem quando o canal declara a capacidade.

### Filtros e métricas
Filtro por canal na lista de tickets e no Dashboard; relatórios não somam canais com regras diferentes sem rotular.

### Acessibilidade e tokens
Só `lucide-react` e tokens semânticos (ADR 0008): ícone de marca de terceiros entra como SVG em `src/assets/channels/`,
nunca emoji. Cores de canal viram tokens (`channel-whatsapp`, `channel-telegram`...) em `theme/tokens/semantic.ts`.

## Alternativas descartadas

- **Um microsserviço por canal (como o engine do WhatsApp).** Faz sentido para o WhatsApp (conexão persistente, UDP,
  sessão). Os outros cinco são **HTTP + webhook**: um adaptador dentro do `business` basta e evita cinco serviços novos
  para operar. O engine continua sendo só o do WhatsApp.
- **Tabela por canal** (`TelegramConnections`...). Multiplicaria controllers, RBAC e migrations. `ChannelConnection` com
  `config` JSON cifrado cobre.
- **Identificar o contato só pelo telefone** (padrão atual). Falha em Telegram, Instagram e TikTok, que não entregam telefone.
- **Reescrever o WhatsApp primeiro.** Arriscado e sem ganho: ele vira adaptador por último, atrás da paridade.

## Verificação

Testes de contrato **compartilhados** que **todo** adaptador roda (inbound idempotente, dedup, regra de resposta,
limite de texto, erro de credencial) contra Postgres e RabbitMQ reais; teste de regressão de que o WhatsApp mantém o
mesmo comportamento após cada fase; e verificação local com o stack (`docs/dev/local-verification.md`).

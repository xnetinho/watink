# ADR 0040: Chave de conversa com prefixo do canal

## Status
Aceito (8/nov/2026)

## Contexto
`Ticket.conversationKey` diz "esta é a mesma conversa" dentro de uma conexão (ADR 0033 e 0038). O índice único de ticket aberto é
`(tenantId, channelConnectionId, conversationKey)`.

Os dois primeiros canais nasceram com formatos diferentes, sem regra:
- o Telegram, o primeiro canal, gravou o `chat_id` **cru** (`29494932`);
- o WhatsApp, ao ganhar a chave na migração `Whatsapps` → `ChannelConnections` (decisão D4, 7/nov), recebeu `wa:<id do contato>`.

A chave só colidia entre canais por sorte: ela é única **por conexão**, então `29494932` do Telegram nunca encontrou `wa:...`. Mas um número cru não diz de que
canal é, e os canais planejados (Mercado Livre `pack_id`, OLX `listId:chatId`, Instagram IGSID) também usam números e textos que **podem coincidir** entre si.

## Decisão
1. **Toda chave de conversa começa com o prefixo do canal**, no formato `<prefixo>:<identificador do canal>`. O prefixo é curto, minúsculo e fixo.
2. **A chave é sempre montada por uma função do `domain`**, nunca por concatenação solta no adaptador. Quem a lê para falar com a plataforma usa a função
   inversa e **recusa** a chave que não é do seu canal (nunca manda um texto qualquer à API como se fosse um destino).
3. **A plataforma externa nunca vê o prefixo.** O adaptador traduz de volta para o identificador puro (por exemplo, a Bot API do Telegram recebe o `chat_id` sem `tg:`).
4. **O identificador depois do prefixo é o da CONVERSA, não o da pessoa.** Em chat privado do Telegram os dois coincidem; em grupo, supergrupo e canal o `chat_id`
   é outro (negativo) e quem escreve (`from.id`) é a pessoa, guardada na `ContactIdentity`.

## Tabela de canais

| Canal | Prefixo | Chave | Função em `domain` | Estado |
|---|---|---|---|---|
| WhatsApp | `wa:` | `wa:<id do contato>` (o contato unifica telefone e LID) | `WhatsAppConversationKey` | em uso |
| Telegram | `tg:` | `tg:<chat_id>` (negativo em grupo, supergrupo e canal) | `TelegramConversationKey`, `TelegramChatID` | em uso |
| Mercado Livre | `ml:` | `ml:<pack_id>` (ou `ml:<order_id>`); pergunta de anúncio `ml:q:<question_id>` | a criar com o adaptador | planejado |
| OLX | `olx:` | `olx:<listId>:<chatId>` | a criar com o adaptador | planejado, bloqueado por aprovação |
| Instagram | `ig:` | `ig:<IGSID>` | a criar com o adaptador | planejado |
| TikTok | `tt:` | `tt:<id da conversa>` | a criar com o adaptador | planejado |

Os formatos de Mercado Livre, OLX, Instagram e TikTok são **propostas** derivadas dos planos em `openspec/changes/add-channel-*`; confirmar com a API real antes de fixar.

## Como adicionar um canal novo
1. Escolher o prefixo e acrescentar a linha na tabela acima **no mesmo PR** do adaptador.
2. Criar `<Canal>ConversationKey(...)` em `domain/channel.go` e, se o adaptador precisar do identificador puro para enviar, a inversa (como `TelegramChatID`).
3. `ParseInbound` preenche `InboundMessage.ConversationKey` com a função, nunca com o valor cru.
4. `Send` traduz a chave de volta e recusa chave de outro canal.
5. Teste: a chave tem o prefixo, ida e volta, e a inversa rejeita chave de outro canal e lixo (modelo: `domain/conversation_key_test.go` e `adapter_send_test.go` do Telegram).

## Consequências
- Positivas: a chave se lê sozinha ao depurar; nenhuma colisão possível entre canais, mesmo se uma lógica futura juntar tickets de canais diferentes (histórico do mesmo
  cliente, carteira, distribuição de fila); canal novo segue um molde.
- Custo: trocar a chave de um canal que já tem ticket aberto faz a próxima mensagem abrir **outro** ticket. Como o projeto não tem produção nem legado (CLAUDE.md), vale a
  regra de sempre: nada de migração incremental; o banco do `lt` é recriado.
- Não muda: o índice único, a busca `FindOpenByConversation`, nem o id da mensagem (`StorageMessageID`).

## Ticket novo é achado e criado pela chave, nunca pelo contato (8/nov/2026)
`FindOrCreatePending` (o caminho que cria o ticket pendente de toda mensagem e chamada recebida) filtrava por **(contato, conexão, pendente)** e ignorava a
chave. Onde o contato **não é** a conversa, a mensagem caía no ticket de outra conversa. Visto no `lt` com o bot real: a mensagem de um grupo do Telegram
(contato = quem escreveu, conversa = o grupo, `tg:-575994051`) foi gravada no ticket **privado** do remetente (`tg:29494932`).

O WhatsApp só escapava porque a chave dele (`wa:<id do contato>`) é calculada a partir do contato; o Telegram privado escapava porque `chat_id = user_id`. Nenhuma
regra impedia um canal novo de ter chave independente do contato, e é o caso dos planejados: **Mercado Livre** (uma conversa por pedido, `pack_id`) e **OLX**
(uma por anúncio, `listId:chatId`) teriam o mesmo comprador com várias conversas e um só contato, e elas seriam **fundidas**.

Regra: o repositório procura por `(tenantId, channelConnectionId, conversationKey)` e `status IN ('open','pending')`, o mesmo critério do índice único
`idx_tickets_open_conversation`, e **recusa** (`ErrConversationKeyRequired`) ticket sem chave. Todo canal novo precisa montar a chave pela função do `domain` (acima)
antes de criar ticket. O teste de contrato `TestFindOrCreatePending_MesmoContatoEmDuasConversasSaoDoisTickets` cobre o caso sem depender de um canal real.


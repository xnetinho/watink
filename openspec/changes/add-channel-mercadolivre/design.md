# Design

Depende de `add-channel-abstraction`. Fonte: documentação do Mercado Livre lida em out/2026 (pós-venda atualizada em
27/04/2026; notificações em 14/09/2026).

## Conexão (OAuth do vendedor)

Authorization Code do ML; `credentialsEnc = {access_token, refresh_token, expires_at, user_id}`; renovação **antes** de
expirar, com lock por conexão (dois refresh simultâneos invalidam o token anterior). Falha de refresh → `status=reauth`
e aviso na UI ("reconectar Mercado Livre"). **Pendente de confirmação:** validade dos tokens e exigência de aprovação.

## Duas famílias de conversa

| | Pós-venda | Pergunta de anúncio |
|---|---|---|
| Recurso | `/messages/packs/{pack}/sellers/{seller}` | `/questions`, `/answers` |
| `conversationKey` | `pack_id` (ou `order_id` se nulo) | `question_id` |
| Quem inicia | comprador | comprador |
| Resposta | texto ≤ 350 + anexos | resposta única por pergunta |
| Contexto | pedido (item, status, envio) | anúncio (item) |

## Capacidades

```go
Capabilities{ InitiateConversation:false, ReplyWindow:48h úteis (bloqueio), MaxTextBytes:350 (ISO-8859-1),
  Media:[jpg,png,pdf,txt ≤25MB], Groups:false, Typing:false, ReadReceipts:true(date_read), Auth:oauth, HasPhone:false }
```

## Agente de IA (mudança de 02/02/2026, MLB)

Para o Brasil, ao **enviar**, `to.user_id` = ID do agente (`3037675074`) e, ao **ler**, `from.user_id` das mensagens do
comprador também é o do agente. Consequências:
- o **remetente real** não vem em `from.user_id`: identificar o comprador pelo **pedido** (`/orders/{id}` → `buyer.id`),
  não pela mensagem;
- **1 mensagem por vez**: a fila de saída serializa por conversa;
- o ID do agente é **configuração** (muda por país e pode mudar), com teste de contrato.

## Entrega e reparo

Webhook (`messages` com `created`/`read`, `questions`) traz só o `resource`; o adaptador faz o `GET` do recurso. O ML
exige **HTTP 200 em até 500 ms**: o handler grava o evento em fila e responde; o processamento é assíncrono. Eventos
perdidos (retentativa por 1 h) são recuperados por `/missed_feeds` (retém 2 dias) em um job de reparo.
Dedup por `message.id`. `GET` de mensagens **marca como lidas**: usar `mark_as_read=false` na leitura de reparo.

## Mapeamento

| ML | Watink |
|---|---|
| `message.id` (hex) | `externalMessageId` |
| `pack_id` | `conversationKey` (e `Ticket.conversationKey`) |
| `buyer.id` (do pedido) | `ContactIdentity(mercadolivre, buyerId)`, `displayName` = apelido |
| `message.text` | corpo; `message_moderation.reason` → aviso |
| `message_attachments` | mídia (baixar para `mediastore`; a chave de upload expira em 48 h) |
| `order` (item, status, envio) | `ticket.context` (cabeçalho: pedido, item, status) |

## Pontos de atenção

- Mensagens do **comprador moderadas não são visíveis**; as do vendedor sim: sempre tratar como "pode faltar".
- A mensageria é bloqueada em `cancelled`, durante mediação e em Full ainda não entregue: devolver motivo legível.
- **Telefone/e-mail do comprador**: a doc cita `/questions/{id}?api_version=4`, mas **o conteúdo não foi confirmado**.

## Verificação

Testes de contrato + servidor ML simulado (OAuth, webhook, `/missed_feeds`, 403 de bloqueio, anexo expirado, agente de
IA, 350 caracteres/charset). Validação ao vivo exige conta de vendedor real: **não há sandbox de mensageria
documentado**; planejar com uma conta de teste e pedido real de baixo valor.

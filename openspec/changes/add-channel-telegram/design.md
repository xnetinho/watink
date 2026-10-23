# Design

Depende de `add-channel-abstraction` (fases 0 a 4). Fonte: documentação oficial (Bot API 10.3, 24/08/2026).

## Conexão

1. O cliente cola o token (`123456:ABC...`). O sistema chama `getMe` para validar e obter `id`/`username`.
2. Guarda `credentialsEnc = {token}` (cifrado) e `config = {botId, username, mode: "webhook"|"polling"}`.
3. Modo webhook: `setWebhook(url=<base>/webhooks/channels/telegram/<connectionId>, secret_token=<segredo da conexão>,
   allowed_updates=["message","edited_message","callback_query","my_chat_member"])`. Portas válidas: 443, 80, 88, 8443.
4. Modo polling: laço `getUpdates(timeout=50)` por conexão, com `offset` persistido; só em deployments sem URL pública.
5. Validação do webhook: header `X-Telegram-Bot-Api-Secret-Token` comparado em tempo constante.

## Capacidades

```go
Capabilities{ InitiateConversation:false, ReplyWindow:nil, MaxTextBytes:4096,
  Media:[photo,video,audio,voice,document,location], Groups:true, Typing:true,
  ReadReceipts:false, Reactions:true, Auth:token, HasPhone:false }
```

## Mapeamento

| Telegram | Watink |
|---|---|
| `update.update_id` | dedup do webhook (`update_id` é sequencial e serve para ignorar repetidos e reordenar) |
| `message.chat.id` | `ContactIdentity.externalId` e `conversationKey` (conversa privada) |
| `message.from` (nome, username) | `displayName` / `meta.username` |
| `chat.type` = `group`/`supergroup` | conversa de grupo (`isGroup`), `conversationKey = chat.id` |
| `message.message_id` | `externalMessageId` (único **por chat**, então a chave é `(chat.id, message_id)`) |
| `reply_to_message` | citação (`quotedMsgId`) |
| `photo[]` (várias resoluções) | escolher a maior; baixar via `getFile` (≤ 20 MB; acima disso, servidor Bot API local) |

## Pontos de atenção

- **`file_path` expira**: baixar na chegada e guardar em `mediastore`, nunca guardar a URL do Telegram (contém o token).
- **Privacidade em grupo**: com o modo de privacidade ligado o bot só vê comandos e menções; documentar na tela de conexão.
- **Bot bloqueado pelo usuário** chega como `my_chat_member`: marcar a identidade como `blocked` e mostrar no ticket.
- **429**: respeitar `parameters.retry_after`.
- **Telefone**: `request_contact` não foi verificado nesta leitura; tratar como não confirmado.

## Verificação

Testes de contrato compartilhados + servidor Telegram **simulado** (`httptest`) para `getMe`, `setWebhook`, `getUpdates`,
`sendMessage`, `getFile`, 429 e `my_chat_member`. Validação ao vivo com um bot de teste real (grátis, sem aprovação).

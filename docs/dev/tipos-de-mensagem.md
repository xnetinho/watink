# Tipos de mensagem recebidos e "mensagem não suportada"

## Como cada tipo chega ao chat
`Messages.mediaType` decide a bolha. O corpo (`body`) de alguns tipos tem formato fixo, lido pelo frontend:

| mediaType | WhatsApp | Telegram | body |
|---|---|---|---|
| chat | texto | texto | o texto |
| image / video / audio / document / sticker | sim | sim (foto, vídeo, vídeo-nota, animação, áudio, voz, documento, sticker estático) | legenda; arquivo em `mediaUrl` |
| location | localização e localização ao vivo | localização / local | `imagem\|link\|descrição` (3 partes, `\|` do nome é trocado por `/`) |
| vcard | contato (1 contato) | contato | vCard (`FN:`, `TEL:`) |
| poll | enquete | enquete | texto formatado com as opções (não há voto aqui) |
| event | evento | não existe | `📅 nome` |
| view_once | visualização única (chega só o aviso) | não existe | aviso |
| **unsupported** | qualquer conteúdo que não conhecemos | qualquer campo que não conhecemos | **nome do tipo** (ex.: `interactiveMessage`, `dice`) |

## Regra de "não suportado" (a prova de futuro)
- **WhatsApp (engine):** `unwrap` abre envelopes (temporária, documento com legenda, visualização única, edição). O que sobra é classificado; se restar algum campo do protobuf fora de `handledOrTechnical`, vira `unsupported` com o nome do campo. Tipo novo que o WhatsApp criar **não some**: aparece como aviso.
- **Telegram (adapter):** campo da mensagem fora de `telegramKnownFields` vira `unsupported`.
- **Frontend:** `mediaType` fora de `KNOWN_MEDIA_TYPES` (ou `unsupported`) mostra o aviso "Mensagem não suportada" com o tipo. Um tipo novo do backend nunca aparece como bolha vazia.
- Prévia do ticket: `⚠️ Mensagem não suportada`.

## Ao suportar um tipo novo
1. Trate-o em `engine-go/.../message_content.go` (ou no adapter do canal) e adicione o campo a `handledOrTechnical`/`telegramKnownFields`.
2. Adicione o `mediaType` em `KNOWN_MEDIA_TYPES` e desenhe a bolha em `MessageMedia.tsx`.
3. Um teste por tipo (há exemplos em `message_content_types_test.go` e `adapter_types_test.go`).

## Atenção
- `LocationPreview` NÃO pode usar HTML: nome e endereço do local vêm de quem enviou.
- Enquete: o voto continua não sendo mostrado no chat (o engine publica `message.poll_vote`, mas só quando a enquete foi criada pelo Watink).

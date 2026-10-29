# Emojis: conjunto de imagens (estilo WhatsApp Web)

## Decisão
O Watink desenha emoji com **imagens do conjunto Apple**, não com a fonte do sistema. O WhatsApp Web usa um conjunto próprio e não distribuído, inspirado no da Apple; o da Apple é o mais próximo disponível. Assim o mesmo emoji (😢) fica igual em Windows, macOS, Android e Linux, e o que se escolhe no seletor é idêntico ao que aparece na bolha.

## Onde se aplica
Texto de mensagens e prévias (via `MarkdownWrapper`), reações, nome do contato (lista e cabeçalho) e o seletor (`EmojiPicker`). Qualquer lugar novo que mostre texto livre do usuário deve usar `<Emojified>`.

## Como funciona
- `public/emoji/apple/*.webp`: 3.783 imagens de 64 px (2 KB cada, ~7,4 MB no total), empacotadas no app. **Sem CDN**: funciona on-premise e offline.
- `lib/emoji/emoji.ts`: `splitEmojis` quebra o texto em trechos (bandeira, família com ZWJ, tom de pele e tecla são UM emoji); emoji sem imagem no conjunto volta como texto e a fonte do sistema o desenha.
- `lib/emoji/apple-index.json` e `catalog.json`: gerados do `emoji.json` do pacote. O nome do arquivo tem no mínimo 4 dígitos hex (`0031-fe0f-20e3`, `00a9`).
- O backend serve `/emoji/*` com cache imutável e **404** quando falta (antes, qualquer arquivo ausente voltava a página inicial com 200).

## Licença (decisão do dono)
O pacote `emoji-datasource-apple` é MIT, mas as **imagens são arte registrada da Apple**. O uso comercial é uma zona cinzenta de licença; o risco foi aceito explicitamente. Alternativas livres: Google Noto (Apache 2.0) e Twemoji (CC-BY 4.0), com visual mais distante do WhatsApp.

## Atualizar o conjunto
Baixar `emoji-datasource-apple` da versão nova, regenerar `public/emoji/apple` (PNG 64 → WebP q90), `apple-index.json` e `catalog.json`. O seletor lê o catálogo; os testes de `lib/emoji` falham se o índice apontar para imagem inexistente.

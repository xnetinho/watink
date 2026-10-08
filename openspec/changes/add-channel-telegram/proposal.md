# Proposal

## Why

Telegram é o canal **mais fácil e sem burocracia** para sair do WhatsApp: o cliente cria um bot no @BotFather, cola o
token e começa a atender, sem aprovação de plataforma e sem risco de ban (Bot API é oficial). Serve de **prova da
abstração de canais** (`add-channel-abstraction`) com o menor custo de integração.

## What Changes

- Canal `telegram` (Bot API) como adaptador de `domain.Channel`.
- **Conexão por token:** o cliente cola o token; o sistema valida (`getMe`), registra o webhook (`setWebhook` com
  `secret_token`) e mostra o nome do bot.
- **Recepção** por webhook (padrão) com `allowed_updates` restrito; **polling** (`getUpdates`) como alternativa quando o
  deployment não tem URL pública, mutuamente exclusivos por regra da API.
- **Envio** de texto (até 4096 caracteres), foto, vídeo, áudio, voz, documento, localização e teclado inline.
- **Conversas privadas e grupos** (grupo exige o bot como membro e respeita o modo de privacidade).
- Identidade do contato pelo `chat.id` (`ContactIdentity`); **sem telefone** (só se o usuário compartilhar o contato).

### Fora do escopo

- **Conta de usuário via MTProto** (`api_id`/`api_hash`): a própria documentação trata clientes não oficiais como
  suspeitos e bane por spam. Só Bot API.
- **Telegram Business Mode** (bot conectado a conta Business com `business_message`): promissor para atender "como a
  empresa", mas exige conta Premium do dono e não foi verificado a fundo; fica para uma change própria depois.
- Pagamentos, mini apps, canais (broadcast), enquetes e Stars.
- Chamadas e vídeo.

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| O bot só escreve a quem falou com ele primeiro | Baixo | Capacidade `InitiateConversation=false`; a UI não oferece "novo ticket" para Telegram |
| Limites de taxa (~1 msg/s por chat, 20/min em grupo, ~30/s global) | Médio | Fila de saída com token-bucket por conexão e tratamento de `429 retry_after` |
| Token vazado dá controle total do bot | Alto | Cifrado em repouso (`cryptobox`), nunca devolvido nem logado; botão "revogar/rotacionar" |
| `getUpdates` e webhook são exclusivos | Baixo | A conexão grava o modo em uso; trocar de modo chama `deleteWebhook`/`setWebhook` |
| Updates ficam só 24 h no servidor | Médio | Webhook com retentativa do Telegram; reconciliar `pending_update_count` no health da conexão |
| Sem telefone para o CRM | Baixo | Documentar; permitir que o atendente informe o telefone manualmente |

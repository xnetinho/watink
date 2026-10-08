# Tasks

> Depende de `add-channel-abstraction` fases 0 a 4 (interface, entrada única, conexão neutra).

## 0. Decisões
- [ ] 0.1 Confirmar webhook como padrão e polling só sem URL pública
- [ ] 0.2 Confirmar a ordem: Telegram é a primeira integração (valida a abstração)

## 1. Backend
- [ ] 1.1 Cliente HTTP da Bot API (`getMe`, `setWebhook`, `deleteWebhook`, `getUpdates`, `getFile`, `send*`) com
      timeouts, retry e tratamento de `429 retry_after`
- [ ] 1.2 Adaptador `telegram` (`Connect`, `Disconnect`, `Send`, `ParseInbound`) e `Capabilities`
- [ ] 1.3 Webhook com `X-Telegram-Bot-Api-Secret-Token` (comparação em tempo constante) e dedup por `update_id`
- [ ] 1.4 Modo polling com `offset` persistido e líder único por conexão (como o scheduler do FlowBuilder)
- [ ] 1.5 Mídia: baixar na chegada para `mediastore`; nunca persistir URL com token
- [ ] 1.6 Grupos: `isGroup`, modo de privacidade documentado, `my_chat_member` (bloqueado/removido)
- [ ] 1.7 Fila de saída com token-bucket por conexão (~1 msg/s por chat)
- [ ] 1.8 Rotação/revogação de token sem perder a conexão

## 2. Frontend
- [ ] 2.1 Assistente "colar token" com validação em tempo real e nome do bot
- [ ] 2.2 Ícone/token de cor `channel-telegram`, badge no ticket, traduções pt/en/es
- [ ] 2.3 Esconder "novo ticket"/chamadas para conversas do Telegram; contador de 4096

## 3. Verificação
- [ ] 3.1 Testes de contrato + servidor Telegram simulado (todos os cenários acima)
- [ ] 3.2 Mutação nos testes de segurança (secret token, dedup, token cifrado)
- [ ] 3.3 Validação ao vivo com um bot real, no stack local por túnel (`docs/dev/local-verification.md`)
- [ ] 3.4 Documentação de usuário (`docs/user/channels/telegram.md`) e `/qa-analyst`

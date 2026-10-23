# Tasks

> Depende de `add-channel-abstraction` fases 0 a 4. **Bloqueada** até a tarefa 0.1 ter resposta positiva da OLX.

## 0. Bloqueios (descobrir ANTES de implementar)
- [ ] 0.1 Contatar `suporteintegrador@olxbr.com`: um SaaS multiempresa consegue registrar uma aplicação de chat? Há
      homologação, contrato ou custo? **Sem resposta positiva, esta change não prossegue**
- [ ] 0.2 Confirmar se o webhook tem assinatura (a doc não cita) e a lista atual de IPs de saída
- [ ] 0.3 Confirmar limite de caracteres, mídia e janela de resposta com uma conta real
- [ ] 0.4 Confirmar a inconsistência da doc (Leads x Chat) sobre responder pelo CRM

## 1. Backend
- [ ] 1.1 Cliente HTTP da OLX (OAuth, ativar/desativar webhook, `chat/send`) com token-bucket **global** (5.000/min por IP)
- [ ] 1.2 OAuth com refresh e `status=reauth`
- [ ] 1.3 Webhook `POST /webhooks/channels/olx/:connectionId/:secret`, allowlist de IP configurável, dedup por `messageId`
- [ ] 1.4 Adaptador `olx` (`ParseInbound`, `Send`, `Connect`/`Disconnect` ativando/desativando o webhook)
- [ ] 1.5 `conversationKey = listId:chatId`; ignorar `origin=seller` e `senderType=system`
- [ ] 1.6 Preencher `Contact` (nome, e-mail) e telefone da identidade a partir do webhook; sugerir vínculo com Cliente
- [ ] 1.7 Fuso do `messageTimestamp` (assumir `America/Sao_Paulo`, registrado no design)

## 2. Frontend
- [ ] 2.1 Botão "Conectar com OLX" (OAuth) e estado "reconectar"
- [ ] 2.2 Cabeçalho do ticket com o anúncio (`listId` + link) e telefone/e-mail do comprador
- [ ] 2.3 Composer só texto, sem anexo, com o aviso de que mídia não é suportada; token `channel-olx`; traduções

## 3. Verificação
- [ ] 3.1 Contrato + servidor OLX simulado
- [ ] 3.2 Mutação nos testes de segurança (segredo da URL, allowlist, dedup, token cifrado)
- [ ] 3.3 Validação ao vivo com conta de anunciante (depende de 0.1)
- [ ] 3.4 Documentação de usuário e `/qa-analyst`

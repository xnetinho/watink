# Tasks

> Depende de `add-channel-abstraction` fases 0 a 4.

## 0. Bloqueios (descobrir ANTES de implementar)
- [ ] 0.1 Registrar o app no Mercado Livre e ler as regras de "Developer Partner Program"/"Validações e requisitos de
      segurança": existe aprovação ou verificação de empresa? Validade do token OAuth?
- [ ] 0.2 Confirmar com conta real se `/questions/{id}?api_version=4` devolve nome, e-mail e telefone do comprador
- [ ] 0.3 Confirmar se há sandbox/conta de teste para mensageria pós-venda
- [ ] 0.4 Decidir se perguntas de anúncio entram na primeira entrega ou só o pós-venda

## 1. Backend
- [ ] 1.1 Cliente HTTP do ML com rate limit (500 rpm por pool) e tratamento de 429/403 com motivos legíveis
- [ ] 1.2 OAuth: authorize, callback, refresh com lock por conexão, `status=reauth`
- [ ] 1.3 Adaptador `mercadolivre`: `ParseInbound` (webhook → `GET` do recurso), `Send` (pós-venda e `POST /answers`)
- [ ] 1.4 `resolveRecipient` isolando o **ID do agente de IA** (configurável) e envio serializado por conversa
- [ ] 1.5 `conversationKey = pack_id`; ticket por pedido; contexto do pedido (`/orders/{id}`)
- [ ] 1.6 Webhook: 200 imediato + fila; dedup por `message.id`; job de reparo por `/missed_feeds`
- [ ] 1.7 Validação de 350 caracteres e charset ISO-8859-1 no servidor, com a lista de caracteres recusados
- [ ] 1.8 Anexos: upload, associação em até 48 h, download para `mediastore`
- [ ] 1.9 Contagem de 48 h úteis e `replyPolicy` (bloqueada, expira em X), alerta de SLA

## 2. Frontend
- [ ] 2.1 Botão "Conectar com Mercado Livre" (OAuth) e estado "reconectar"
- [ ] 2.2 Cabeçalho do ticket com o pedido (nº, item, status, envio) e link para o pedido
- [ ] 2.3 Composer: contador 350, charset, anexo JPG/PNG/PDF/TXT, motivo do bloqueio e prazo restante
- [ ] 2.4 Distinguir pós-venda de pergunta de anúncio; aviso de moderação; traduções; token `channel-mercadolivre`

## 3. Verificação
- [ ] 3.1 Contrato + servidor ML simulado (todos os cenários de risco acima)
- [ ] 3.2 Mutação nos testes de segurança (OAuth, dedup, token cifrado)
- [ ] 3.3 Validação ao vivo com conta de vendedor, por túnel (ver 0.3)
- [ ] 3.4 Documentação de usuário e `/qa-analyst`

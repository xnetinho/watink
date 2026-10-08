# Proposal

## Why

Para quem vende no Mercado Livre, **a conversa com o comprador é o atendimento**: pós-venda (`/messages/packs`) e perguntas
de anúncio (`/questions`). Hoje o vendedor responde no site do ML, fora do Watink, sem fila, sem histórico unificado e
sem fluxo. Trazer isso para o Watink dá fila, SLA, pipeline e CRM ao atendimento de marketplace.

É também o canal que **mais testa a abstração**: a conversa é de um **pedido**, não de uma pessoa, e há regras
restritivas que o produto precisa tornar visíveis.

## What Changes

- Canal `mercadolivre` como adaptador de `domain.Channel`, com **dois tipos de conversa**: **pós-venda** (por `pack_id`)
  e **pergunta de anúncio** (por `question_id`/`item_id`).
- **Conexão por OAuth** do vendedor; token e refresh cifrados; renovação automática e estado "reconectar".
- **Recepção por webhook** (tópicos `messages`, `questions`) com **reparo por `/missed_feeds`** (guarda 2 dias).
- **Envio**: resposta pós-venda (texto até **350** caracteres, ISO-8859-1; anexos JPG/PNG/PDF/TXT até 25 MB) e **resposta
  de pergunta** (`POST /answers`).
- **Uma conversa por pedido** (`conversationKey = pack_id`, ou `order_id` quando o pack é nulo) e contexto do pedido no
  cabeçalho do ticket.

### Fora do escopo
- Iniciar conversa com o comprador: **a API não permite** (só o comprador inicia).
- Reclamações e mediações (`/claims`), devoluções e envio de notas fiscais: outros recursos do ML, outra change.
- Publicar/editar anúncios, estoque, preços.
- Outros países além do **MLB** (Brasil) nesta primeira entrega; MLC/MLA/MLM entram depois.

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| **Agente de IA no meio (desde 02/02/2026, MLB):** `to.user_id` do envio deve ser o ID do **agente** (`3037675074`) e o `from.user_id` das leituras também; só **1 mensagem por vez** | **Alto** | Isolar em um único ponto (`resolveRecipient`), com o ID do agente **configurável** e teste de contrato; a doc diz "sem novos endpoints", mas o comportamento mudou |
| **48 h úteis** para resolver antes de a conversa ser **bloqueada** (`blocked_conversation_send_message_forbidden`) | **Alto** | Capacidade `ReplyWindow` por conversa, contagem regressiva no composer, alerta de SLA e fila prioritária |
| Texto de **350** caracteres em **ISO-8859-1** | Médio | Contador no composer, validação no servidor (caractere fora do charset recusado com a lista) |
| **Moderação** de mensagens (bloqueia links de redes sociais, dados pessoais, Mercado Pago/PayPal) | Médio | Mostrar o `moderation.reason` no ticket; mensagem do comprador moderada **não aparece** na API (documentar) |
| Envio bloqueado em ordem `cancelled`, em mediação e em Full não entregue | Médio | Mapear 403 para motivos legíveis; desabilitar o composer com o motivo |
| Webhook responde em **500 ms** ou é reenviado por 1 h; depois vai a `/missed_feeds` (2 dias) | Médio | Responder 200 imediato e processar em fila; job de reparo periódico |
| Rate limit 500 rpm GET e 500 rpm POST/PUT (pools separados) | Baixo | Token-bucket por vendedor |
| **Aprovação do app / validade do token / validação de empresa: não confirmados** | Médio | Registrar o app e ler as regras de "Developer Partner Program" **antes** de implementar; ver tarefa 0.2 |

# Desenho: fila de distribuição de tickets (WhatsApp e Telegram)

Status: **proposta para revisão**. Nada daqui foi implementado. Escopo inicial: WhatsApp e Telegram. Mercado Livre, OLX, Instagram e TikTok entram depois,
pelo mesmo ponto de entrada (ADR 0033 e 0040).

## 1. O que existe hoje (lido no código em 10/nov/2026)

| Ponto | Hoje |
|---|---|
| Estratégia da fila | `Queues.distributionStrategy`: `MANUAL`, `AUTO_ROUND_ROBIN`, `AUTO_BALANCED`; `prioritizeWallet` (carteira primeiro). |
| Quando a distribuição roda | **Só** quando uma pessoa troca a fila de um ticket na tela (`update_ticket.go`). |
| Ticket novo de mensagem | Nasce `pending`, sem dono. A fila vem de `resolveChannelQueue`: só quando a conexão está ligada a **exatamente 1** fila (`receive_message.go`). O Telegram nem isso: o ticket nasce sem fila. |
| Ticket novo de chamada | Nasce `pending`; fila só se a conexão tem 1 fila (`calls/events.go`). Quem toca = `Eligible`: vê a conexão, tem `calls:receive`, está online e não pausou. |
| Nó `ticket` do bot | Cria ticket sem passar pela distribuição. |
| Grupo | Nasce `open` e sem dono (WhatsApp e, desde o #182, Telegram). Distribuição não se aplica. |
| Visibilidade do operador | Um OU de 3 condições (`pkg/auth/tenant.go`): ticket atribuído a ele, **ou** da fila dele, **ou** de conexão ligada a uma fila dele. |
| Carteira | `Contacts.walletUserId`. Só é usada se a fila tem `prioritizeWallet`. |
| Presença | Só existe para chamada (navegador conectado ao SSE). Mensagem não considera quem está online. |

Consequência: a distribuição automática **existe, mas quase nunca é acionada**. E o ticket atribuído a A continua visível a B se B está na mesma fila.

## 2. Princípios (decisões já tomadas)

1. **Simples e modular.** O administrador escolhe uma estratégia; não existe um número fixo de "tipos" de fila.
2. **Não reaproveita o FlowBuilder.** O fluxo é atendimento automatizado (bot); distribuição a humanos é outro domínio.
3. **Mensagem e chamada seguem a mesma estratégia.** Mudar isso numa só não pode divergir da outra.
4. **Por canal agnóstico.** A regra não conhece WhatsApp nem Telegram: recebe "fatos neutros" do canal (seção 5).
5. **Privacidade comercial.** Vendedor não vê cliente atendido por outro (comissão). Suporte técnico precisa de agilidade (ver o ticket e assumir rápido).

## 3. Proposta: um ponto de entrada e regras em ordem

### 3.1 Ponto de entrada único
Toda criação de ticket (mensagem, chamada, nó `ticket` do bot, ticket manual) passa por **um** serviço `TicketRouting.Route(ticket, fatos)`. Hoje cada caminho decide sozinho.
Ele devolve uma decisão e a explica:

```
decisão = { fila, dono (ou ninguém), quem é notificado, o que fazer se ninguém responder, motivo }
```

### 3.2 Regras em ordem ("se isto, então aquilo")
A fila tem uma lista **ordenada** de regras; vale a primeira que casar. Cada regra responde 3 perguntas:

| Pergunta | Opções (módulos de estratégia) |
|---|---|
| **Quem é o dono?** | carteira do contato; rodízio; menos tickets abertos; ninguém (fica na fila); pessoa fixa. |
| **Quem é notificado?** | só o dono; todos da fila; os online da fila. |
| **E se ninguém responde em N minutos?** | nada; escalar para a próxima regra; avisar o supervisor; soltar o dono e voltar à fila. |

Cada opção é um módulo pequeno atrás da mesma interface (`Strategy`). Acrescentar uma nova não mexe nas outras. As atuais (`MANUAL`, `AUTO_ROUND_ROBIN`, `AUTO_BALANCED`, carteira) viram os primeiros módulos, sem mudar o comportamento.

### 3.3 Explicação por ticket
Cada decisão grava "por quê" (qual regra, qual módulo, quem foi candidato) no log do ticket (`TicketLogs`, `LogType: route`). O supervisor consegue responder "por que caiu para o João?".

### 3.4 Modelos prontos antes do construtor livre
Entrega inicial com 3 modelos que o administrador seleciona, sem montar regra:

- **Vendas (carteira):** carteira do contato; se não houver, rodízio entre os online; ticket só visível ao dono (ver 4).
- **Suporte (agilidade):** todos da fila são notificados; o primeiro que assume vira o dono; ticket visível à fila toda.
- **Manual:** como hoje (ninguém atribuído, a fila decide).

O construtor livre de regras vem depois, só se os modelos não bastarem.

## 4. Visibilidade (segurança)
Muda o OU de 3 condições. Proposta, por fila, um campo **visibilidade**:
- `fila` (padrão atual): todos da fila veem.
- `dono`: enquanto o ticket tem dono, só ele (e supervisor/gerente) vê; sem dono, a fila vê.

É a mudança de maior risco: toca `pkg/auth/tenant.go`, e precisa de teste com RLS real (`rls_session_test.go`) e prova por mutação. Por isso fica **por último**.

## 5. Fatos neutros do canal (adaptador)
A regra não lê o canal diretamente. O adaptador entrega:

| Fato | WhatsApp | Telegram |
|---|---|---|
| Identidade da pessoa | contato (telefone/LID) | `ContactIdentity` (user_id) |
| Chave da conversa | `wa:<contato>` | `tg:<chat_id>` |
| É grupo? | sim/não (grupo nasce `open`, fora da fila) | idem (#182) |
| Urgência | chamada de voz = alta; mensagem = normal | mensagem = normal |
| Contexto | conexão, fila(s) da conexão | conexão, fila(s) da conexão |

Telegram não tem telefone, então **carteira** só funciona se o contato já tiver `walletUserId` (definido por quem atende ou por vínculo com cliente do CRM).

## 6. Chamada de voz
Mesma decisão de dono/notificação do ticket. Diferença: o toque só vai a quem está **online** (já é a regra de `Eligible`). Se ninguém está online, o ticket de chamada perdida cai na fila e segue a mesma regra de "ninguém responde".

## 7. Plano em etapas (cada uma um PR, mensurável)
1. **Ligar o que existe:** `Route` chama a `DistributeTicketUseCase` atual na entrada de mensagem, chamada e nó `ticket`. Sem regra nova. Teste: ticket novo de uma fila `AUTO_*` já nasce com dono.
2. **Telegram herda a fila da conexão** (`resolveChannelQueue` no `channel_inbound`), igual ao WhatsApp.
3. **Explicação por ticket** (`route` no `TicketLogs`).
4. **Notificação e escalonamento** ("ninguém responde em N minutos").
5. **Modelos prontos** (Vendas, Suporte, Manual) na tela da fila.
6. **Visibilidade `dono`** com testes de RLS e mutação.
7. Construtor livre de regras, só se necessário.

## 8. Decisões que preciso de você
1. **Sem dono e sem carteira:** cai na fila inteira (suporte) ou rodízio (vendas)? Proposta: depende do modelo.
2. **Ausência:** quantos minutos sem resposta antes de escalar? Proposta: configurável por fila, padrão 5 min.
3. **Quem é supervisor** (vê tudo e recebe o escalonamento)? Proposta: perfil que já tem `alcance` de empresa; ou um papel novo?
4. **Telegram sem carteira:** aceitar rodízio puro, ou exigir vínculo com cliente do CRM antes?
5. **Etapa 1 agora?** É a de menor risco e já corrige o defeito mais visível (ticket novo nunca é distribuído).

## 9. Fora de escopo
Mercado Livre, OLX, Instagram e TikTok (dependem de aprovações externas). Priorização por SLA. Distribuição por habilidade/idioma. Mídia e áudio não mudam.

## 10. Riscos
- Etapa 1 muda comportamento de produção para quem já tem fila `AUTO_*` configurada: tickets que hoje ficam sem dono passarão a ter dono.
- Visibilidade `dono` pode esconder ticket de quem hoje o vê; precisa de aviso na tela e migração de dados só no `lt`.
- Escalonamento exige um relógio (varredura periódica), como o `DeliveryWatchdog`.

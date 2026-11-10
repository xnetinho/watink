# Pesquisa: como outras ferramentas distribuem conversas (e o que isso muda no nosso desenho)

Data: 10/nov/2026. Fontes: documentação pública de Chatwoot, Intercom, Zendesk, Freshdesk/Freshchat e Octadesk (lista no final). Tudo abaixo vem dessas páginas.

## 1. O padrão que se repete

Em todas as ferramentas, o modelo é o mesmo, e é **pequeno**:

> **Caixa** (equipe/inbox/grupo) → **uma estratégia** (manual, rodízio, balanceado) → **elegíveis** (membros + online/disponíveis) → **um limite de capacidade** → quem não cabe **espera na própria caixa**.

| Ferramenta | Unidade de roteamento | Estratégias | Elegibilidade | Capacidade | Sem ninguém disponível |
|---|---|---|---|---|---|
| **Chatwoot** | Inbox (caixa de entrada) | Rodízio e Balanceado | Membro do inbox **e** com status disponível (offline/ocupado são pulados) | Limite por agente e por inbox (política de capacidade) | Fica **não atribuída** na caixa até alguém liberar |
| **Intercom** | Caixa da equipe (team inbox) | Manual, Rodízio, Balanceado | Membro, ativo; "ausente" é pulado (opção por equipe) | Limite por pessoa e, no plano maior, por caixa | Espera na caixa; a tela mostra o **motivo** ("todos no limite") |
| **Zendesk** | Fila (omnichannel) | Capacidade disponível ou rodízio | Status unificado + capacidade + (opcional) habilidade | Regras de capacidade por canal | Fila padrão; pode reatribuir ticket não aceito |
| **Freshdesk / Freshchat** | Grupo | Rodízio, Carga (load), Habilidade | Agentes ativos no roteamento | Limite por agente | Aguarda; regras de atribuição têm precedência sobre a automática |
| **Octadesk** | Grupo de atendimento + "Responsável pelo contato" | Roleta; carteira por contato | Grupo + horário | Máximo de atendimentos simultâneos por grupo | Cliente com responsável **sempre** vai para ele, mesmo offline ou no limite |

## 2. O que eles fazem que nós ainda não temos (vale copiar)

1. **Estratégia por caixa, só três opções.** Manual, Rodízio, Balanceado. Nenhuma ferramenta pesquisada tem "regras em ordem com 3 perguntas" por caixa.
2. **Elegibilidade = membro + disponível.** Quem está offline ou ausente é pulado. É a nossa Q9, e todas as ferramentas fazem por padrão, não como opção por fila.
3. **Capacidade é um único número.** Chatwoot e Intercom: máximo de abertas por agente. Quando todos estão no limite, o ticket **espera na caixa**. É mais simples que nosso escalonamento.
4. **Motivo visível quando espera.** O Intercom mostra "todos no limite" ou "nenhum elegível" no próprio ticket. É a nossa Q29, só que bem menor que gravar uma explicação completa.
5. **Destino padrão para o que não casou.** Intercom: "default assignee" (pode ser uma equipe ou ninguém). Zendesk: fila padrão. É a nossa fila padrão (F1), confirmada como padrão do mercado.
6. **Carteira ("responsável pelo contato") vence tudo.** Octadesk: cliente com responsável vai **sempre** para ele, mesmo offline ou no limite. Também é "privacidade": o vendedor não vê cliente que tem responsável. Isso **contradiz a nossa Q12** (carteira offline segue a regra normal) e casa com a regra de comissão que você descreveu.
7. **Excluir usuário ou caixa pede um destino.** Octadesk: ao excluir o usuário, escolhe quem recebe **toda** a carteira. Intercom: ao excluir a caixa, escolhe a quem reassociar conversas, regras e mensagens. É a nossa Q18/Q19.
8. **Horário por caixa, e só avisa.** Chatwoot e Octadesk: horário por caixa/fila, com **mensagem de ausência** automática uma vez por dia por conversa, suprimida se um agente respondeu nos últimos 5 min. O Chatwoot não bloqueia a entrada nem muda o status: **a conversa chega normalmente**.

## 3. O que eles **não** fazem (o que podemos cortar)

| Item do nosso desenho | O que o mercado faz | Recomendação |
|---|---|---|
| Fila com **tipo** (humana/fluxo/recado) | Ninguém tem. Horário = **mensagem de ausência** + o bot/fluxo é outro recurso (workflow/automação) | **Cortar o tipo.** Fora do horário: mensagem de ausência por fila. O fluxo continua sendo disparado como hoje (por conexão e palavra-chave) |
| Regra de roteamento com critério (Q3) | Intercom tem regras, mas **fora** da caixa (workflow que escolhe a equipe); dentro da caixa só há as 3 estratégias | Separar: **escolher a fila** (poucas regras simples, ou o bot) e **escolher a pessoa** (3 estratégias) |
| Escalonamento para fila pai (Q15, Q25) | Zendesk: "reatribuir se não aceito". Chatwoot/Intercom: nada automático; o ticket só espera | **Cortar a hierarquia.** Se for preciso: "sem resposta em N min, volta à fila" (um só comportamento) |
| Visibilidade por dono configurável (Q22) | Octadesk: **uma regra**: cliente com responsável fica com ele; agente vê "os meus + os sem responsável" | **Uma regra**, não duas por fila |
| Estratégias novas (Q8: primeiro que assume, pessoa fixa) | Só rodízio e balanceado; "pessoa fixa" = carteira | **Só 3**: manual, rodízio, balanceado; carteira como exceção por cima |
| Último atendente configurável por fila (Q14) | Octadesk: "responsável" é um campo do contato, definido automático no 1º atendimento | **Um campo no contato** (já existe `walletUserId`), sem lógica por fila |
| Online configurável por fila (Q9) | Sempre só disponíveis | **Sempre só online**; remove um campo |
| 36 decisões, 10 PRs | — | Cabe em 4 PRs (seção 5) |

## 4. O que a pesquisa aponta que **também** precisamos decidir (não estava no desenho)

- **Capacidade por agente** (Chatwoot e Intercom): um número. Eu tinha deixado como "Q10: fora desta fase". O mercado trata como **parte do núcleo**: sem ele o balanceamento não protege ninguém e o ticket nunca "espera na caixa".
- **Status do agente** (disponível / ausente / ocupado): todos têm. Nós só temos pausa para chamada. É o que torna a elegibilidade real.
- **Reabertura**: Zendesk e Intercom reabrem para o mesmo agente se ele está disponível, senão voltam à caixa. É a nossa Q14, mais simples.

## 5. Desenho mínimo proposto (substitui o consolidado da Parte J)

**Conceitos (3):**
1. **Fila** = grupo de pessoas + **estratégia** (`manual` | `rodizio` | `balanceado`) + **limite por pessoa** (opcional) + **mensagem de ausência** com **horário** (opcional).
2. **Fila padrão** da empresa: destino de quem não casou (confirma F1).
3. **Responsável do contato** (`walletUserId`, já existe): vence a estratégia, mesmo offline ou no limite.

**Comportamento (1 caminho, para mensagem, chamada e bot):**
1. Grupo? Fica fora (como hoje).
2. Contato tem responsável? Vai para ele.
3. Senão: elegíveis = membros da fila, **online**, **fora de pausa**, **abaixo do limite**; aplica a estratégia.
4. Ninguém elegível? **Espera na fila**, com o motivo visível no ticket ("todos no limite", "nenhum online").
5. Fora do horário? Envia a mensagem de ausência uma vez por dia por conversa; o ticket entra normalmente.
6. Conexão sem fila usa a fila padrão; sem fila padrão, só o admin vê, com aviso (já decidido).

**Cortado:** tipo de fila, `flowId`, fila de recado/fluxo, hierarquia e escalonamento para pai, visibilidade configurável por fila, estratégias extras, "último atendente" por fila, "online configurável".

**Entrega (4 PRs em vez de 10):**
1. **Integridade**: excluir fila/usuário pede destino (mover) e não deixa ticket órfão (Q18, Q19, Q20).
2. **Ponto único + fila padrão + Telegram herda a fila**, ligando a distribuição que já existe (Q1 a Q7).
3. **Disponibilidade e capacidade**: online, pausa para mensagem, limite por pessoa, "espera na fila" com motivo, atribuição transacional (Q9, Q10, Q11, Q32, Q29).
4. **Horário e mensagem de ausência por fila** + **painel** com "sem fila" e "sem dono" (Q31, Q34).

**Fica para depois, só se a operação pedir:** escalonamento, visibilidade "só o dono", fila pai, tipos de fila.

## 6. Pontos em que a pesquisa contradiz uma decisão sua (a confirmar)

| Decisão atual | O que o mercado faz | Pergunta |
|---|---|---|
| **Q12:** carteira offline segue a regra normal | Octadesk: o responsável **sempre** recebe, mesmo offline ou no limite | Para vendas (comissão), o cliente deve esperar o dono voltar? |
| **Q4:** visibilidade só pela fila do ticket | Octadesk: agente vê "os meus + os sem responsável" | Vendedor vê ticket sem dono da fila dele e os próprios, e nenhum de outro vendedor? |
| **Q15/Q25:** subir para a fila pai | Ninguém escala por hierarquia; o ticket espera ou é reatribuído à própria caixa | Basta "sem resposta em N min, volta à fila"? |
| **F4:** fila com tipo `fluxo`/`recado` | Ninguém; usam mensagem de ausência e automação separadas | Mensagem de ausência por fila resolve o que você queria? |

## 7. O que não verifiquei
- Li documentação pública, não usei as ferramentas. Zendesk e Freshdesk têm planos com funções pagas que a documentação mistura; usei só o que está descrito como padrão ou núcleo.
- Não pesquisei Blip, Zenvia nem Take: a busca em português retornou só Octadesk com conteúdo útil.
- "Octadesk: carteira vence tudo" vem de uma FAQ de um plano de 2023; o comportamento pode variar por plano.

## Fontes
- Chatwoot: Assignment Policies, Agent Capacity Policies, Business Hours and Auto-Responder, Lesson 4 (Automation and Routing).
- Intercom: Organize team inboxes, Workload management explained, Round robin / Balanced assignment deep dive, Inbox Assignment Limits, Assign conversations to teammates and teams.
- Zendesk: Omnichannel routing (configuração, filas, regras de capacidade, status unificado), Channel Attributes API.
- Freshworks: Configure Freshdesk Omniroute, What is IntelliAssign in Freshchat.
- Octadesk: "O que é o Responsável pelo contato", "Como crio um Bot", artigo de fila de atendimento.

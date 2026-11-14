# Fila de distribuição: reanálise profunda e questões de decisão

Status: **relatório para decisão**. Complementa e corrige `desenho-fila-de-distribuicao.md` (que cobria só a entrada do ticket). Nada aqui foi implementado.
Data da análise: 10/nov/2026. Tudo marcado como **fato** foi lido no código ou medido no banco do `lt`; o que é **proposta** está dito.

## Parte A. O que a primeira proposta deixou de fora

A primeira proposta tratava só de "como um ticket novo escolhe um dono". A reanálise achou 14 áreas que, se ficarem de fora, viram débito técnico:

| # | Área | Estado hoje (fato) |
|---|---|---|
| 1 | Conexão sem fila | Ticket nasce sem fila e sem dono; operador comum não o vê (só admin); ninguém é avisado. Chamada de voz não toca para ninguém. A criação da conexão não exige fila. No `lt`, 5 de 5 tickets estão sem fila. |
| 2 | Conexão com várias filas | `resolveChannelQueue` devolve `nil` com 0 **ou** várias filas. |
| 3 | Ciclo de vida do ticket | Fechado e o cliente volta: nasce **outro** ticket, sem herdar fila nem dono. Não há "último atendente" gravado. |
| 4 | Excluir fila | Bloqueado pelo banco (FK sem `ON DELETE`) **se** há usuário ou conexão vinculados; **sem** vínculo, a exclusão passa e o ticket fica com `queueId` apontando para uma fila inexistente. Medido no `lt` (em transação, revertida). |
| 5 | Excluir ou desativar usuário | `Tickets.userId` tem FK sem `ON DELETE` e o `user_mutation` só limpa `UserSetor`. Não há regra para os tickets dele. |
| 6 | Hierarquia de filas | `Queues.parentId` existe, é usado só para ordenar a lista. Nada herda ou escala por ele. |
| 7 | Setor (Acessos) | Existe `Setores` + `setor_filas` + `UserSetor.ehGestor`, **paralelo** a `user_queues`. A visibilidade do operador (`tenant.go`) **não usa** setor. Há dois modelos de "quem pertence a quê". |
| 8 | Nó `ticket` do bot | `moveToQueue` grava `queueId` e `status=pending`; `assignUser` grava `userId` e `open`. Ambos escrevem direto, sem passar por nenhuma regra. |
| 9 | Transferência manual | `update_ticket.go` distribui só quando a fila **muda**; transferir para o mesmo usuário ou para a mesma fila não aciona nada. |
| 10 | Presença, pausa e capacidade | Presença e pausa existem **só para chamada**. Não há limite de tickets por operador nem status "disponível" para mensagem. |
| 11 | Horário de atendimento | Não existe por fila nem por conexão. |
| 12 | Notificação ao operador | Existe `NotificationsPopOver` e `notify.ts` no frontend; não há notificação de "ticket novo na sua fila" por regra. |
| 13 | Grupos | Nascem `open`, sem dono e fora da fila (WhatsApp e Telegram). A visibilidade do grupo para operador comum depende da conexão ligada a uma fila dele. |
| 14 | Métricas e auditoria | `TicketLogs` registra `status`, `assign` e `transfer`; não há "por que caiu para ele". `dashboard.go` agrega por fila (ignora ticket sem fila). |

## Parte B. Questões para decisão

Cada questão tem a **recomendação** e o **custo de não decidir**. Marque a opção ou escreva a sua.

### Bloco 1: Destino do ticket sem fila (a mais urgente)

**Q1. Conexão sem fila: para onde vão os tickets?**
- **A.** Fila padrão da empresa (uma fila marcada como "padrão"; conexão sem fila usa essa). *Recomendada.*
- **B.** Caixa de triagem: estado "sem fila", visível ao supervisor, com aviso.
- **C.** Exigir pelo menos uma fila para ativar a conexão.
- **D.** Manter como está (só o administrador vê).
- Custo de não decidir: tickets (e chamadas) invisíveis para operador comum, sem alerta. Hoje é o estado de 100% dos tickets do `lt`.

**Q2. Se a empresa não tem fila padrão (e a conexão não tem fila), o que acontece?**
- **A.** A tela de Conexões mostra aviso vermelho e o ticket fica visível só ao administrador. *Recomendada* (mínimo).
- **B.** Bloqueia a conexão até haver fila.
- **C.** Cria automaticamente uma fila "Geral" no primeiro ticket.

**Q3. Conexão com várias filas: como escolher a fila do ticket novo?**
- **A.** A primeira regra de roteamento que casar; se nenhuma casar, a fila padrão. *Recomendada.*
- **B.** Sempre a fila padrão, e o operador transfere.
- **C.** Rodízio entre as filas da conexão.
- **D.** O bot de triagem decide (como hoje com `nil`), sem regra do sistema.

**Q4. Quem pode ver ticket de conexão que está ligada a **várias** filas?** Hoje o operador de **qualquer** uma delas vê (OU de 3 condições).
- **A.** Manter. *Recomendada no início.*
- **B.** Só quem é da fila do ticket (depois que ele recebe fila).

### Bloco 2: Ponto de entrada e quem decide

**Q5. Um ponto de entrada único para todo ticket novo?** Hoje cada caminho decide sozinho (mensagem, chamada, bot, ticket manual, webchat, campanha de grupos, encaminhar).
- **A.** Sim: `TicketRouting.Route` para todos. *Recomendada.*
- **B.** Só mensagem e chamada; os demais ficam como estão.
- Custo de B: o bot e o ticket manual continuam sem regra e o desenho "mensagem e chamada seguem a mesma estratégia" fica pela metade.

**Q6. O nó `ticket` do bot (`moveToQueue`/`assignUser`) deve passar pela regra?**
- **A.** Sim: `moveToQueue` aciona a distribuição da fila de destino; `assignUser` é uma escolha explícita e **respeita** (não redistribui). *Recomendada.*
- **B.** Sempre escreve direto (como hoje).

**Q7. Webchat e campanha de grupos criam ticket sem passar pelo serviço de entrada.** Entram no ponto único?
- **A.** Sim. *Recomendada.*
- **B.** Webchat sim; campanha de grupos não (grupo não tem fila).

### Bloco 3: Estratégias e regras

**Q8. Quais estratégias entram na primeira entrega?**
- **A.** As 3 que existem (`MANUAL`, `AUTO_ROUND_ROBIN`, `AUTO_BALANCED`) + carteira, sem comportamento novo. *Recomendada na etapa 1.*
- **B.** Acrescentar já: "primeiro que assume", "online primeiro", "pessoa fixa".

**Q9. Rodízio e balanceamento consideram quem está online e disponível?** Hoje não: sorteiam entre **todos** da fila, mesmo desconectados.
- **A.** Só entre quem está online (mesma `Presence` da chamada). Se ninguém está, cai na fila. *Recomendada.*
- **B.** Todos, como hoje.
- **C.** Configurável por fila.

**Q10. Limite de tickets abertos por operador?**
- **A.** Não nesta fase. *Recomendada.*
- **B.** Sim, por fila (ex.: máx. 10); cheio, pula para o próximo.
- Custo de A: o balanceamento mede só a contagem; um operador sobrecarregado continua recebendo se for o "menos pior".

**Q11. O operador pode "pausar" o recebimento de tickets?** Hoje a pausa vale só para chamada.
- **A.** Sim, a mesma pausa vale para mensagem. *Recomendada.*
- **B.** Só chamada.

**Q12. A carteira do contato (`walletUserId`) vale quando o dono da carteira está offline ou pausado?**
- **A.** Sim, o ticket vai para ele mesmo assim (cliente é dele; comissão). *Recomendada para Vendas.*
- **B.** Não, segue a regra normal.
- **C.** Depende do modelo (Vendas = A; Suporte = B).

**Q13. Telegram não tem telefone: como definir a carteira?**
- **A.** Só por vínculo manual de quem atende ou por vínculo com cliente do CRM; sem vínculo, rodízio. *Recomendada.*
- **B.** Exigir vínculo antes de distribuir.

### Bloco 4: Ciclo de vida

**Q14. Cliente volta depois de fechar: quem atende o ticket novo?**
- **A.** O último atendente (precisa gravar "último atendente"), se ele ainda está na fila e disponível. *Recomendada para Vendas.*
- **B.** Segue a regra normal (como hoje).
- **C.** Configurável por fila.

**Q15. Ticket aberto sem resposta do operador por N minutos (escalonamento):**
- **A.** Configurável por fila; ação padrão: avisar o supervisor. *Recomendada.*
- **B.** Soltar o dono e voltar à fila.
- **C.** Escalar para a fila pai (`parentId`).
- **D.** Nada.
- Pergunta embutida: o prazo conta de quê? **(i)** da chegada, **(ii)** da última mensagem do cliente sem resposta.

**Q16. Ticket `pending` há muito tempo sem ninguém aceitar:**
- **A.** Mesmo escalonamento da Q15. *Recomendada.*
- **B.** Nada.

**Q17. Transferir um ticket para outra fila:**
- **A.** Sempre redistribui na fila nova (se a estratégia for automática). *Recomendada.*
- **B.** Mantém o dono atual se ele também é da fila nova.
- **C.** O operador escolhe na hora ("para a fila" ou "para a pessoa").

### Bloco 5: Exclusões e integridade (débito técnico real)

**Q18. Excluir uma fila que tem ticket aberto, usuário ou conexão vinculados:**
- **A.** Bloquear com mensagem clara ("mova os N tickets e remova os vínculos antes"). *Recomendada.*
- **B.** Perguntar para qual fila mover os tickets e os vínculos, e mover.
- **C.** Soltar os tickets (`queueId = NULL`) e remover os vínculos.
- Fato: hoje, sem usuário nem conexão vinculados, a exclusão **passa** e deixa ticket com `queueId` apontando para uma fila que não existe. Isso precisa de uma correção independente do resto do desenho (chave estrangeira ou regra).

**Q19. Excluir ou desativar um usuário que tem tickets abertos:**
- **A.** Bloquear até transferir. *Recomendada para exclusão.*
- **B.** Devolver os tickets à fila (soltar o dono) e redistribuir. *Recomendada para desativação.*
- **C.** Reatribuir ao gestor do setor (`ehGestor`).

**Q20. Remover um usuário de uma fila enquanto ele tem tickets dela:**
- **A.** Os tickets continuam com ele até fechar. *Recomendada.*
- **B.** Voltam para a fila.

**Q21. Desvincular uma conexão da última fila dela:**
- **A.** Avisa e aplica a Q1 (fila padrão) aos próximos tickets. Os existentes não mudam. *Recomendada.*
- **B.** Bloqueia.

### Bloco 6: Visibilidade, setor e hierarquia

**Q22. Visibilidade do ticket com dono (a decisão de comissão):**
- **A.** Por fila: todos da fila veem (hoje). Padrão para Suporte.
- **B.** Por dono: só o dono e o gestor veem enquanto ele tem dono; sem dono, a fila vê. Padrão para Vendas.
- **C.** Escolha por fila, com os dois modelos prontos. *Recomendada.*

**Q23. Quem é "gestor" para ver tudo e receber o escalonamento?** Existem dois conceitos: o `alcance` do usuário (empresa/plataforma, vê tudo) e o `ehGestor` do setor.
- **A.** `ehGestor` do setor vê os tickets das filas **do setor dele** e recebe o escalonamento delas. Quem tem `alcance` de empresa vê tudo. *Recomendada.*
- **B.** Só `alcance`; o setor não participa.
- **C.** Criar um papel novo de supervisor.

**Q24. Setor e `user_queues` são dois modelos paralelos de pertencimento. O que fazer?**
- **A.** `user_queues` continua sendo a **fonte única** de visibilidade e distribuição; o setor só agrupa filas para gestão e relatório. *Recomendada.*
- **B.** Setor passa a conceder visibilidade (usuário do setor vê as filas do setor).
- **C.** Unificar os dois num só modelo (mudança maior, fora desta fase).

**Q25. Hierarquia de filas (`parentId`) passa a ter significado?**
- **A.** Não nesta fase (continua só para ordenar). *Recomendada.*
- **B.** Sim: o supervisor da fila pai vê as filhas e o escalonamento sobe para ela.

### Bloco 7: Grupos, canais e chamadas

**Q26. Grupo (WhatsApp e Telegram) entra na distribuição?**
- **A.** Não: nasce `open`, sem dono, fora da fila (como hoje). *Recomendada.*
- **B.** Entra na fila da conexão e pode ter dono.
- Pergunta embutida: grupo de uma conexão sem fila é visível para operador comum? Hoje depende do mesmo OU de 3 condições; sem fila na conexão, só o administrador.

**Q27. Chamada de voz recebida:**
- **A.** Mesma decisão de dono e notificação do ticket; toca só para quem está **online**, e a pausa vale. Ninguém online: vira chamada perdida no ticket da fila, com o escalonamento da Q15. *Recomendada.*
- **B.** Como hoje (toca para todos os elegíveis).

**Q28. Canais novos (Mercado Livre, OLX, Instagram, TikTok):**
- **A.** Entram pelo mesmo ponto de entrada, sem lógica própria de distribuição. *Recomendada.*
- Pergunta embutida: o Mercado Livre tem **uma conversa por pedido**. O mesmo comprador com dois pedidos cai no mesmo dono (carteira/último atendente) ou na regra normal?

### Bloco 8: Operação, auditoria e migração

**Q29. Explicação por ticket ("por que caiu para o João?"):**
- **A.** Gravar a decisão no `TicketLogs` (`route`: regra, módulo, candidatos, motivo) e mostrar no histórico do ticket. *Recomendada.*
- **B.** Só log técnico, sem tela.

**Q30. Notificação de ticket novo para o operador:**
- **A.** Notificar o dono (ou os online da fila, se não há dono) pelo `NotificationsPopOver` e som existente. *Recomendada.*
- **B.** Sem notificação nova.

**Q31. Horário de atendimento por fila:**
- **A.** Fora desta fase; deixar o gancho (a regra recebe "agora está dentro do horário?"). *Recomendada.*
- **B.** Entra já (mensagem automática fora do horário, fila fechada).

**Q32. Concorrência: duas mensagens chegam ao mesmo tempo para uma fila em rodízio.**
- **A.** Atribuição **transacional** com trava por fila, para o rodízio não dar o mesmo dono duas vezes nem ultrapassar a vez. *Recomendada* (sem isso o rodízio é racional só em baixa carga).
- **B.** Sem trava (como hoje).

**Q33. Tickets que já existem sem fila e sem dono quando a regra for ativada:**
- **A.** Não mexer nos existentes; vale só para os novos. *Recomendada.*
- **B.** Rodar uma redistribuição única, sob botão do administrador, com prévia.

**Q34. Métricas e painel:**
- **A.** Mostrar "tickets sem fila" e "tickets sem dono há mais de N min" no painel. *Recomendada.*
- **B.** Fora desta fase.

**Q35. Onde configurar?**
- **A.** Na própria tela da fila (campos: estratégia, carteira, visibilidade, escalonamento) e na de Conexões (fila padrão e aviso). *Recomendada.*
- **B.** Tela nova de "Roteamento".

### Bloco 9: Ordem de entrega

**Q36. Ordem proposta (cada item é um PR, com teste de RLS real e prova por mutação):**
1. **Integridade (independente do resto):** excluir fila/usuário sem deixar ticket órfão (Q18, Q19, Q20).
2. **Destino do ticket sem fila:** fila padrão e aviso na tela de Conexões (Q1, Q2, Q3).
3. **Ponto de entrada único** ligando a distribuição existente a mensagem, chamada, bot e ticket manual (Q5 a Q8).
4. **Telegram herda a fila da conexão** (igual ao WhatsApp).
5. **Atribuição transacional** (Q32) e **online/pausa** (Q9, Q11).
6. **Explicação por ticket** e notificação (Q29, Q30).
7. **Escalonamento** (Q15, Q16).
8. **Último atendente** e **carteira no Telegram** (Q13, Q14).
9. **Visibilidade por dono** e papel do gestor do setor (Q22 a Q24), por último por ser a de maior risco.
- **A.** Concordo com a ordem.
- **B.** Quero outra ordem (qual?).
- **C.** Quero que 1 e 2 saiam antes de qualquer outra coisa, em paralelo.

## Parte C. Riscos que atravessam tudo
- **Mudança de comportamento em produção** (etapa 3): quem tem fila `AUTO_*` passa a ter tickets novos com dono.
- **Visibilidade por dono** pode esconder ticket de quem hoje o vê. Precisa de aviso na tela e de teste com RLS real (`rls_session_test.go`) e mutação.
- **Escalonamento** exige um relógio (varredura periódica, como o `DeliveryWatchdog`), com trava distribuída para não rodar em dobro com mais de uma réplica.
- **Presença** hoje só conhece quem tem navegador aberto (SSE): usá-la para mensagem esconde quem usa só o celular. Decidir na Q9.
- **Sem produção nem legado:** colunas e tabelas novas entram direto no modelo GORM/baseline; no `lt` crio à mão ou recrio o banco.

## Parte D. O que não foi verificado
- Não executei nenhum cenário novo de distribuição; só li o código e consultei o banco do `lt`.
- O `lt` tem 1 usuário (administrador, `alcance=plataforma`) e nenhum vínculo `user_queues`: o comportamento para operador comum foi lido no SQL de `tenant.go`, **não** exercitado com um operador real.
- Não medi a carga real de mensagens por fila; a Q32 é uma análise do código, não de um problema observado.

## Parte E. Decisões tomadas (10/nov/2026)

| Q | Decisão |
|---|---|
| 1 | Conexão sem fila: **mantém como está** (só o administrador vê). Existe uma **fila padrão opcional**; sem ela vale este comportamento. |
| 2 | Sem fila padrão e sem fila na conexão: **aviso vermelho** na tela de Conexões. |
| 3 | Conexão com várias filas: **primeira regra de roteamento que casar; senão a fila padrão** (se existir). |
| 4 | Ticket de conexão com várias filas: **só a fila do ticket vê** (sai o "conexão ligada a fila do usuário" do OU de visibilidade). |
| 5 | **Ponto de entrada único** para todo ticket novo. |
| 6 | Nó `ticket` do bot passa pela regra: `moveToQueue` aciona a distribuição; `assignUser` é escolha explícita e é respeitada. |
| 7 | Webchat e campanha de grupos **entram** no ponto único. |
| 8 | Estratégias já na primeira entrega: as atuais + carteira **e as novas** ("primeiro que assume", "online primeiro", "pessoa fixa"). |
| 9 | Considerar só quem está online: **configurável por fila**. |
| 10 | Sem limite de tickets por operador nesta fase. |
| 11 | A **pausa** de recebimento vale também para mensagem. |
| 12 | Carteira com dono offline ou pausado: **não vale**; segue a regra normal. |
| 13 | Telegram: carteira por vínculo manual ou CRM; sem vínculo, rodízio. |
| 14 | Cliente que volta: **configurável por fila** (último atendente ou regra normal). |
| 15 | Sem resposta por N min: **subir para a fila pai** (prazo contado da última mensagem do cliente sem resposta). |
| 16 | `pending` parado: mesmo escalonamento da Q15. |
| 17 | Transferir: **o operador escolhe** ("para a fila" ou "para a pessoa"). |
| 18 | Excluir fila: **mover** tickets e vínculos para outra fila escolhida. |
| 19 | Excluir ou desativar usuário: **devolver os tickets à fila sempre** (solta o dono e redistribui). |
| 20 | Sair de uma fila: os tickets **continuam** com o usuário até fechar. |
| 21 | Desvincular a última fila da conexão: avisa; próximos tickets seguem Q1/Q2; os existentes não mudam. |
| 22 | Visibilidade de ticket com dono: **escolha por fila** (por fila, ou por dono). |
| 23 | Gestor: `ehGestor` do setor vê e recebe o escalonamento das filas do setor; `alcance` de empresa vê tudo. |
| 24 | `user_queues` é a **fonte única** de visibilidade e distribuição; o setor só agrupa filas. |
| 25 | **Fila pai vê as filas filhas** (a hierarquia `parentId` passa a ter significado). |
| 26 | Grupo **não** entra na distribuição. |
| 27 | Chamada de voz segue a **mesma regra** do ticket (toca só para online; pausa vale; sem ninguém online vira chamada perdida com escalonamento). |
| 28 | Canais novos entram pelo mesmo ponto; pedidos do mesmo comprador seguem o mesmo dono (carteira ou último atendente). |
| 29 | **Explicação** da decisão: grava no `TicketLogs` e mostra no histórico. |
| 30 | Notificar o dono, ou os online da fila quando não há dono, pelo `NotificationsPopOver` e som existentes. |
| 31 | **Horário de atendimento entra**, e fora do horário deve haver **roteamento** (atendimento automatizado ou recado). Detalhes na Parte F. |
| 32 | Atribuição **transacional** com trava por fila. |
| 33 | Tickets existentes: **não mexer**. |
| 34 | Painel mostra "sem fila" e "sem dono há mais de N min". |
| 35 | Configuração nas **telas existentes** (fila e Conexões). |
| 36 | **Ordem proposta**: integridade, fila padrão, ponto único, Telegram herda fila, transacional e online/pausa, explicação e notificação, escalonamento, último atendente e carteira, visibilidade por dono. |

## Parte F. Consequências e conflitos que as decisões criam (a fechar antes do código)

1. **Q4 remove uma condição da visibilidade.** Hoje o operador vê o ticket se for dele, da fila dele, **ou** de conexão ligada a uma fila dele. Com a Q4, sai a terceira. Passa a valer: ticket só é visível se tem fila (ou dono). Ticket sem fila (Q1) fica só para o administrador. Exige que **todo** ticket receba fila pela regra da Q3 ou pela fila padrão.
2. **Q15 não diz o que fazer quando a fila não tem pai.** Precisa de um destino de reserva.
3. **Q31 é uma funcionalidade nova grande.** Pede horário por fila (dias, horas, fuso, feriados) e um destino fora do horário (fluxo automatizado ou recado). É o único item que muda o escopo.
4. **Q19 solta o dono, mas o ticket pode não ter fila.** Devolver "à fila" precisa de uma fila; sem ela o ticket fica sem dono e sem fila.
5. **Q18 (mover para outra fila) e Q25 (pai vê filhas) interagem:** excluir uma fila pai precisa decidir o destino das filhas.
6. **Q9 e Q14 por fila** exigem campos novos no cadastro da fila, e a tela da fila (Q35) cresce.

## Parte G. Decisões das consequências (10/nov/2026)

| # | Decisão |
|---|---|
| F1 | **Fila padrão obrigatória na prática**: sem regra que case, o ticket usa a fila padrão. Sem fila padrão, fica sem fila e só o administrador vê (com o aviso vermelho da Q2). |
| F2 | Fila **sem pai** que não é respondida: **soltar o dono e voltar à fila** (outro operador pode assumir). Com pai, sobe para a fila pai (Q15). |
| F3 | **Horário de atendimento** entra como **etapa própria depois do escalonamento**, sem atrasar as anteriores. |
| F4 | Destino fora do horário: **outra fila**, a critério do administrador. O administrador sugeriu que **"fluxo automatizado" e "recado" sejam tipos de fila**; assim o horário só precisa apontar para uma fila (ver Parte H). |

## Parte H. Consequência da F4: fila com tipo

Se fluxo automatizado e recado são **tipos de fila**, a fila deixa de ser só "um grupo de atendentes". Passa a ter um `tipo`:

| Tipo | O que é | Tem atendentes? | Estratégia de distribuição? |
|---|---|---|---|
| `humana` (hoje) | grupo de operadores | sim | sim |
| `fluxo` | o ticket entra num fluxo automatizado (FlowBuilder) | não | não |
| `recado` | envia uma mensagem fixa e deixa o ticket aguardando | não | não |

Isso resolve o horário sem criar conceito novo: "fora do horário, a fila X aponta para a fila Y (do tipo fluxo ou recado)". Também conversa com a decisão já tomada de **não reaproveitar o FlowBuilder para a distribuição**: a distribuição continua separada; o fluxo é só um **destino** que a fila pode ter.

## Parte I. Decisões sobre o tipo de fila (10/nov/2026)

| # | Decisão |
|---|---|
| H1 | A fila do tipo `fluxo` aponta para um fluxo por um campo **`flowId`** na fila; ao entrar nela, o ticket dispara esse fluxo. |
| H2 | A fila do tipo `recado` envia o texto configurado **uma vez por ticket** e deixa o ticket **aguardando** o próximo expediente. |
| H3 | **Volta ao expediente**: no início do horário, os tickets das filas de recado/fluxo voltam à fila humana de origem e entram na distribuição. |
| H4 | O tipo `fluxo` é o **mesmo mecanismo** para triagem dentro do expediente: qualquer fila pode ser o destino padrão de uma conexão. |

## Parte J. Desenho consolidado (todas as decisões aplicadas)

### Modelo de dados (proposta; sem produção nem legado, entra direto no modelo GORM)
- **`Queues`**: `tipo` (`humana` | `fluxo` | `recado`), `padrao` (boolean, no máximo uma por empresa), `flowId` (tipo fluxo), `recadoTexto` (tipo recado), `somenteOnline` (Q9), `retornoCliente` (`ultimo_atendente` | `regra`, Q14), `visibilidade` (`fila` | `dono`, Q22), `escalonaApos` (minutos, Q15/Q16), `horario` (jsonb: dias, faixas, fuso) e `foraDoHorarioFilaId` (F4).
- **`Tickets`**: `ultimoAtendenteId` (Q14), `ultimaMensagemClienteEm` (prazo do escalonamento, Q15).
- **`TicketLogs`**: tipo `route` com a decisão (regra, módulo, candidatos, motivo; Q29).
- Remoção de condição na visibilidade (`pkg/auth/tenant.go`): sai "conexão ligada à fila do usuário" (Q4), entra "fila pai vê as filhas" (Q25) e `ehGestor` por setor (Q23).

### Fluxo de um ticket novo (`TicketRouting.Route`)
1. Grupo? Nasce `open`, sem fila e sem dono (Q26). Fim.
2. Descobre a fila: regra de roteamento da conexão → fila padrão (F1) → sem fila (só admin).
3. Fora do horário da fila? Vai para `foraDoHorarioFilaId` (tipo `fluxo` ou `recado`; F3/F4/H1/H2).
4. Tipo `humana`: aplica a estratégia da fila (carteira só se o contato tem `walletUserId` e a fila pede; último atendente se `retornoCliente`; rodízio/balanceado/primeiro que assume/pessoa fixa; só online se `somenteOnline`; pausa vale). Atribuição transacional com trava por fila (Q32).
5. Grava a decisão no `TicketLogs` (Q29) e notifica o dono ou os online da fila (Q30).
6. Escalonamento (varredura periódica): sem resposta em `escalonaApos` min contados da última mensagem do cliente sem resposta → sobe para a **fila pai**; sem pai, **solta o dono e volta à fila** (F2).

### Ordem de entrega (Q36 + F3)
1. Integridade: excluir fila (**mover** tickets e vínculos, Q18), excluir/desativar usuário (**devolver à fila**, Q19), `queueId` órfão.
2. Fila padrão + aviso na tela de Conexões (Q1, Q2, F1).
3. Ponto único `TicketRouting.Route` (Q5, Q6, Q7) ligando a distribuição existente.
4. Telegram herda a fila da conexão.
5. Atribuição transacional (Q32), online configurável (Q9) e pausa para mensagem (Q11).
6. Explicação por ticket e notificação (Q29, Q30).
7. Escalonamento: fila pai e solta o dono (Q15, Q16, F2), com a visibilidade da fila pai (Q25).
8. **Horário e tipos de fila** `fluxo` e `recado` (Q31, F3, F4, H1 a H4).
9. Estratégias novas, último atendente e carteira no Telegram (Q8, Q13, Q14).
10. Visibilidade por dono e `ehGestor` (Q22 a Q24), por último por ser a de maior risco.

### Lacunas que ainda não decidi sozinho
- **Fuso e feriados** do horário: um fuso por fila ou por empresa? Feriados entram?
- **Conexão com regra de roteamento (Q3):** o que é uma "regra" (palavra-chave? tipo de mensagem? contato novo/recorrente?). Hoje não existe nenhum critério para escolher entre filas.
- **Quando o ticket volta do recado ao expediente (H3):** quem faz a varredura e como evita redistribuir um ticket que o cliente já retomou.
- **Fila padrão e Q4:** como o aviso de F1 aparece para o operador comum (que não vê o ticket sem fila).

## Parte K. DECISÃO FINAL (10/nov/2026): desenho mínimo, que substitui as Partes E a J onde divergirem

Depois da pesquisa em Chatwoot, Intercom, Zendesk, Freshdesk e Octadesk (`pesquisa-fila-outras-ferramentas.md`), o desenho foi **simplificado**. As Partes E a J ficam como histórico; vale o que está aqui.

### Confirmações da pesquisa
| # | Decisão |
|---|---|
| Q12 | Carteira: o ticket **vai para o dono mesmo offline ou no limite** (cliente de carteira é dele; comissão). |
| Q4 | Visibilidade: o operador vê **os tickets dele e os sem dono das filas dele**; nunca os de outro operador. Uma regra só, sem escolha por fila. |
| Q15 | Sem resposta em N minutos: o ticket **volta à fila** (solta o dono). **Sem hierarquia.** |
| F4 | Fora do horário: **mensagem de ausência** por fila (uma vez por dia por conversa); o ticket entra normalmente e espera o expediente. |

### O que sai do escopo
Tipo de fila (`fluxo`, `recado`) e `flowId`; hierarquia (`parentId`) e escalonamento para a fila pai (Q15 anterior, Q25); visibilidade configurável por fila (Q22); estratégias extras (Q8: "primeiro que assume", "pessoa fixa"); "último atendente" por fila (Q14); "só online" configurável (Q9: passa a ser sempre).

### O desenho
**Três conceitos:** (1) **fila** = pessoas + estratégia (`manual` | `rodizio` | `balanceado`) + limite por pessoa (opcional) + mensagem de ausência com horário (opcional); (2) **fila padrão** da empresa; (3) **responsável do contato** (`walletUserId`, já existe).

**Um só caminho** para mensagem, chamada, bot, ticket manual, webchat e campanha:
1. Grupo: fica fora (Q26).
2. Contato com responsável: vai para ele.
3. Senão: elegíveis = membros da fila, **online**, **fora de pausa**, **abaixo do limite**; aplica a estratégia.
4. Ninguém elegível: **espera na fila**, com o **motivo visível** no ticket.
5. Fora do horário: mensagem de ausência.
6. Conexão sem fila usa a fila padrão; sem fila padrão, só o administrador vê, com aviso (Q1, Q2, F1).

### Entrega em 4 PRs
1. **Integridade:** excluir fila e usuário pede destino e move; nenhum ticket órfão (Q18, Q19, Q20).
2. **Ponto único + fila padrão + Telegram herda a fila**, ligando a distribuição existente (Q1 a Q7). Implementado em duas partes: **2a** = roteamento único (`internal/routing`), fila padrão da empresa e canais genéricos herdando a fila; **2b** = visibilidade Q4 (meus tickets + sem dono das minhas filas), aplicada ao banco (`GetScopedDB`), ao histórico de chamadas e ao SSE.
3. **Disponibilidade e capacidade:** online, pausa para mensagem, limite por pessoa, espera com motivo e atribuição transacional (Q9, Q10, Q11, Q29, Q32).
4. **Horário e mensagem de ausência** + **painel** "sem fila" e "sem dono" (Q31, Q34).

Fica para depois, só se a operação pedir: escalonamento, visibilidade "só o dono", fila pai.

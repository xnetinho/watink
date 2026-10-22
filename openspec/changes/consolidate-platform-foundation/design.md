# Design

## Context

Medido no código (out/2026), ver `proposal.md` para a motivação:

- **Duas tabelas de conexão** (`Whatsapps` e `ChannelConnections`) com sequências de id independentes (hoje `Whatsapps.id=2`
  e `ChannelConnections.id=1`). O adaptador WhatsApp faz `models.Whatsapp{ID: conn.ID}`, o que só funciona se os ids coincidirem.
- **`whatsappId` em 9 tabelas:** `Tickets`, `Users`, `Flows`, `Assistants`, `CallLogs`, `group_caches`, `group_campaigns`,
  `group_campaign_targets`, `group_campaign_sends`. Tickets de canal gravam `0` (sentinela).
- **`Messages.id` é chave primária global** e recebe o id externo cru; `ON CONFLICT DO NOTHING` descarta mensagens de
  outra conversa ou empresa com o mesmo id (o `message_id` do Telegram é sequencial por chat).
- **Envio:** `message_send.go` consulta `Whatsapps` antes de olhar o canal, então o ramo agnóstico é código morto. Há 16
  construtores manuais de routing key `wbot.%s.%d.%s` e 3 ramos `EngineType != "whatsmeow"` (resíduo do izapia).
- **FlowBuilder** tem registro próprio (`flow.ChannelRegistry`) e o nó de mensagem faz `MustGet("whatsapp")`.
- **Engine:** `defaultPublishEvent` só faz `log.Printf` se a publicação falhar; `main.go` dá `Ack` em todo comando, inclusive
  nos que falharam; `Connect()` reconecta mas `ConsumeCommands` roda uma vez; um único `amqp.Channel` publica e consome; o
  único SQL é `session_loader.go:44` sobre `Whatsapps`; o mapa `clients` é indexado só pelo id da sessão.
- **Contrato duplicado à mão:** `engine-go` e `business` são módulos Go separados, sem `go.work`; o engine publica
  `session.pairingcode` e o business escuta `session.pairing_code`; o business trata `message.poll_vote`, que o engine nunca emite.
- **Autorização:** 80 das 214 rotas protegidas não têm `RequirePermission`, incluindo cerca de 50 mutações.
- **Camadas:** 62 de 82 controllers usam o GORM direto (386 `Raw/Exec`); só 10 usam repositório; zero `recover()` fora de HTTP.
- **VoIP:** `-race` verde; `CallManager` chama `OnStateChange` com o mutex travado; vídeo implementado sem validação ao
  vivo; chamadas modeladas como recurso do WhatsApp (`wbot.*.*.call.*`, `Ticket.WhatsappID`).
- **CI:** o job do engine só faz `build` e `vet`; nenhum workflow usa `-race`; `Dockerfile.business` usa `golang:1.25rc1` e `alpine:latest`.

## Goals / Non-Goals

**Goals:**
- Um canal novo se adiciona sem tocar o núcleo, provado por um canal de teste.
- Nenhuma mensagem, evento ou comando se perde em silêncio.
- Contrato com o engine em fonte única, com falha de CI quando divergir.
- Toda mutação exige permissão declarada.

**Non-Goals:**
- Isolamento por RLS: é a change `enforce-tenant-rls`.
- Compatibilidade com dados ou rotas antigas: não há instância em produção; o banco é recriado.
- Implementar Instagram, Mercado Livre, OLX ou TikTok (continuam em changes próprias, depois desta).
- Escalar o engine horizontalmente: instância única por decisão do dono; o limite é documentado.
- Reescrever o FlowBuilder ou o plugin de grupos além do necessário para usar o serviço de envio.

## Decisions

### D1. Uma tabela de conexões; o `sessionId` do engine é o `connectionId`
`ChannelConnections` passa a ser a única tabela. Campos específicos do WhatsApp (proxy, `keepAlive`, sync, `wid`, risco) vão
para `whatsapp_connection_details` (1:1 por `connectionId`). Todas as 9 referências `whatsappId` viram `connectionId NOT NULL`
com FK.
*Por quê:* sem dado legado, não há razão para traduzir ids entre dois espaços; o sentinela `0` e a FK removida hoje somem juntos.
*Alternativa rejeitada:* manter `Whatsapps` e traduzir `connectionId → sessionId` no adaptador. Funciona com dado
legado, mas mantém dois espaços de id para sempre.

### D2. Mensagem com chave primária interna e unicidade composta
`Messages.id` vira chave interna (`bigserial`); `externalId` e `connectionId` viram colunas com
`UNIQUE (tenantId, connectionId, externalId)`, e o `ON CONFLICT` usa essa chave. O id externo continua exposto ao front como
`externalId` onde ele é necessário (citação, ack, reação).
*Alternativa rejeitada:* id textual composto `<kind>:<conn>:<ext>`. Resolve a colisão, mas mistura identidade e formato, e
não há razão para o remendo sem dado legado. Como primeiro PR isolado e rápido, pode-se aplicar o id composto antes da
refatoração completa (tarefa 1.1 da onda de estanque).

### D3. Contrato do engine em módulo compartilhado
Novo módulo `contracts/wbot` (via `go.work`) com: nomes de comandos e eventos como constantes, payloads tipados,
`wbot.Command(tenant, connection, cmd)` e `wbot.Event(...)` como únicos construtores de routing key, e a versão do contrato no
envelope. Os dois lados importam o módulo; um lint proíbe a string `"wbot."` fora dele.
*Alternativa rejeitada:* gerar código a partir de um esquema (protobuf/JSON Schema). Custo de ferramenta sem ganho para dois
consumidores em Go no mesmo repositório.

### D4. Teste de contrato bidirecional
Um teste enumera as constantes do módulo e prova que (a) todo evento declarado tem emissor no engine e tratador no business,
(b) todo bind do business tem emissor, (c) todo comando consumido pelo engine tem construtor no business. Falha hoje em
`pairingcode` e `poll_vote`.

### D5. Engine resiliente
Dois `amqp.Channel` (publicação e consumo) mais um por consumidor, com supervisor que refaz bind após `NotifyClose`.
Publicação com *publisher confirms* e **saída durável**: eventos vão a uma fila local em disco (`outbox`) com reenvio e
retirada só após confirmação. Comandos: `Ack` só no sucesso, `Nack(requeue=false)` com DLQ e retry limitado nos recuperáveis.
O engine continua "adaptador burro": a classificação recuperável/irrecuperável é por tipo de erro.
*Alternativa rejeitada:* confiar no `log.Printf` e no reenvio do WhatsApp; o WhatsApp não reentrega mensagens recebidas.

### D6. Handshake `engine.ready`
O engine publica `engine.ready` ao subir; o business responde com um `session.start` por conexão ativa (mesmo contrato já
existente). Remove `session_loader.go`, o `time.Sleep(5s)` e o filtro `engineType`. O engine passa a ter acesso ao banco
apenas ao schema `whatsmeow` (ver `enforce-tenant-rls` D10).

### D7. `MessageDispatchService` como único caminho de saída
Serviço em `services/` que: resolve a conexão pelo ticket, valida `ReplyPolicy` e `Capabilities`, resolve o endereçamento
**pelo adaptador** (`Channel.ResolveAddress`), envia, persiste a mensagem de saída com `externalId` e publica o evento.
Controllers (`message_send`, `quick_answer`), `flow`, assistentes, campanhas de grupo e plugins chamam este serviço; o
controller fica com parse e resposta. O método de endereçamento acaba com os 6 construtores de JID e com
`contactJID` no caminho de envio.

### D8. Adaptador WhatsApp real e registro único
O adaptador WhatsApp implementa `Connect`, `Disconnect`, `Send` (com citação, menções e mime), status e ciclo de vida de
verdade, falando com o engine pelo módulo de contrato. Os três ramos `EngineType != "whatsmeow"` e o provider izapia
residual são removidos. `flow.ChannelRegistry` e `OutboundChannelAdapter` são eliminados: o FlowBuilder usa o
`MessageDispatchService`.

### D9. Chamadas e vídeo como capacidades
`Capabilities` ganha `voice` e `video`. Os comandos `call.*` passam a carregar `connectionId`, e a interface de chamadas
deixa de ler `Ticket.WhatsappID`. Vídeo permanece **experimental atrás de flag** até validação ao vivo (decisão do dono); o
ADR 0032 é atualizado para refletir o que existe e o que não foi validado.
Os hooks do `CallManager` passam a ser assíncronos (fila interna), eliminando a armadilha de chamar o hook com o mutex travado.

### D10. Autorização por tabela de rotas
Um teste percorre o roteador do Gin e falha se uma rota autenticada que altera dados não tiver permissão nem estiver no
registro de isenções (arquivo versionado com justificativa). As permissões que faltam
(`contacts:delete_all`, `contacts:bulk_delete`, `messages:send`, `flows:delete` etc.) entram no catálogo.

### D11. Camadas no business
Controllers ficam finos (parse, chamada ao serviço, resposta). O acesso a dados passa a repositórios por empresa, começando
pelos módulos que tocam canais (`message`, `ticket`, `contact`, `quick_answer`). O barramento de eventos em memória
(`go func` por handler) é trocado por *outbox* transacional. Um `safeGo` (com `recover`, log e métrica) substitui todo `go func`,
e o consumidor AMQP contém pânico por entrega.
*Alternativa rejeitada:* reescrever todos os 82 controllers de uma vez. Migra-se por módulo, um PR por vez, com o lint de
arquitetura impedindo que novos controllers acessem o GORM.

### D12. Canal `fake` como prova de agnosticismo
Um canal `fake` no pacote de teste implementa o contrato completo e é exercitado por um teste de ponta a ponta
(conectar, receber, enviar, mídia, status, janela, fluxo, resposta rápida) **sem alterar uma linha do núcleo**. É o critério
objetivo de pronto da change.

### D13. Frontend agnóstico
`useConnections` único, `/connections/:id`, tipo `Connection` com `kind` e `capabilities`; `useWhatsApps` e `/whatsapp/*`
saem. Menções e citações passam por `contactId` e são resolvidas no backend (fim do `@s.whatsapp.net` no cliente). O ticket
serializa `channel` e `capabilities`; composer, lista e cabeçalho reagem à capacidade.

### D14. Operação e CI
Prometheus (`/metrics`) nos dois serviços, prontidão real (banco, broker, cache), `trace_id` do webhook ao SSE. CI roda
`go test -race` nos dois módulos, testes do engine, contrato, arquitetura e portas internas; Dockerfiles com versões fixas.
Instância única é a premissa oficial do engine, do poller do Telegram e do `Migrate` (com advisory lock); o limite fica
documentado em ADR.

### D15. `poll_vote`: emitir o evento no engine
O recurso de enquete com captura de resultados (`capture_results` na resposta rápida) existe no frontend e no business, mas
o engine não emite o voto. Decisão: **implementar a emissão** (whatsmeow oferece `DecryptPollVote`) e cobrir com teste de
contrato; a alternativa de remover o recurso quebraria uma opção já exposta ao usuário.
*Pendente de confirmação do dono (ver Open Questions).*

## Risks / Trade-offs

- **Mudança muito ampla.** Mitigação: um PR por tarefa, ondas ordenadas, cada uma com teste e portão local.
- **Remover `Whatsapps` toca engine, adaptador, 9 tabelas e plugins.** Mitigação: PR de esquema primeiro com testes de
  repositório; o canal `fake` e a suíte de contrato travam regressões.
- **Pareamento de todo WhatsApp de novo** (armazenamento do `whatsmeow` recriado). Aceito pelo dono: sem produção.
- **Saída durável do engine adiciona estado em disco.** Mitigação: tamanho limitado, métrica de profundidade, alerta.
- **Reentrega de evento pode duplicar.** Mitigação: `UNIQUE (tenantId, connectionId, externalId)` e dedup no Redis.
- **Hooks assíncronos do `CallManager` mudam a ordem dos eventos de estado.** Mitigação: fila ordenada por chamada e teste de ordem.
- **Dependência de `enforce-tenant-rls`.** O esquema novo nasce com `tenantId` e política; fazer as duas em ordem evita retrabalho.

## Migration Plan

Banco greenfield, sem retrocompatibilidade. Ordem: (0) estanque e guardrails, que não dependem de decisão de modelo;
(1) contrato e lint; (2) esquema novo (conexão única, mensagem, contato); (3) núcleo de canais e adaptador WhatsApp; (4) camadas
do business e frontend; (5) VoIP e operação; (6) governança. Reverter = reverter o PR da tarefa; não há dado a recuperar.

## Open Questions

- **`poll_vote` (D15):** confirmar com o dono se emitimos o voto no engine ou removemos a captura de resultados.
- **Piso de cobertura de testes** nos pacotes críticos (tarefa 8.3): definir o número com o dono.

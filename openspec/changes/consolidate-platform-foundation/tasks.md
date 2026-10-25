# Tasks

> Fonte: `proposal.md`, `design.md` e as quatro specs. **Um PR por item numerado (ou menor)**, em branch `fix/*` ou `feat/*`,
> nunca direto em `develop`, verificado localmente (`docs/dev/local-verification.md`) **antes** do push e com aprovação
> explícita do dono. Banco greenfield: sem migração de dados nem compatibilidade com instalação anterior.
> Convenções: `Session(NewDB:true)`, testes contra Postgres e RabbitMQ reais, lint v2.12.2 local, `-race`.

## 1. Estanque (independe de decisão de modelo)

- [x] 1.1 Mensagens não colidem: id composto `<kind>:<connectionId>:<externalId>` para canais não-WhatsApp e `externalId` em `Messages`; verificar com teste que dois chats com o mesmo `externalId` gravam as duas mensagens
- [x] 1.2 `RequirePermission` nas mutações hoje sem permissão (mensagens, contatos, filas, fluxos, funis, tags, respostas rápidas, plugins, histórico) e catálogo de permissões novas; verificar com teste por rota que atendente sem permissão recebe 403
- [x] 1.3 Teste de tabela de rotas que percorre o roteador e falha para rota autenticada de escrita sem permissão nem isenção registrada; verificar que falha antes de 1.2 e passa depois
- [x] 1.4 Registro versionado de rotas isentas com justificativa (logout, `/me`, leitura própria); verificar que cada isenção tem texto de justificativa
- [x] 1.5 `safeGo` (recover, log, métrica) em todas as goroutines do business e contenção de pânico por entrega no consumidor AMQP; verificar com teste que injeta pânico num tratador e o processo continua
- [x] 1.6 Engine: corrigir `session.pairingcode`/`pairing_code`; verificar com teste de contrato que o evento chega ao tratador
- [x] 1.7 Engine: `Ack` só no sucesso, `Nack` com fila de falhas e retry limitado; verificar com teste que um envio com erro recuperável é reprocessado e um inválido vai à fila de falhas
- [x] 1.8 Engine: consumidor se refaz após queda do broker (canal por consumidor + supervisor); verificar com teste que derruba e religa o RabbitMQ e o consumo volta sozinho
- [x] 1.9 Engine: saída durável de eventos com confirmação de publicação; verificar com teste que `message.received` emitido com o broker fora é entregue depois, sem duplicar
- [x] 1.10 CI: engine roda `go test` e `-race`, business roda `-race`; pin de `golang` e `alpine` nos Dockerfiles; verificar que o CI falha ao introduzir corrida ou imagem móvel
- [x] 1.11 Teste que reprova porta 8084 ou 8085 publicada em `ports:` nos arquivos de composição; verificar adicionando uma porta indevida e vendo falhar

## 2. Contrato do engine

- [x] 2.1 Módulo `contracts/wbot` com `go.work`: constantes de comandos e eventos, payloads tipados, versão no envelope, `Command(...)` e `Event(...)`; verificar que business e engine compilam importando só o módulo
- [x] 2.2 Migrar os 16 construtores de routing key do business e o roteador do engine para o módulo; verificar com busca que `"wbot."` aparece apenas no módulo
- [x] 2.3 Teste de contrato bidirecional (emissor, tratador, bind, comando); verificar que falha ao remover um tratador e passa restaurado
- [x] 2.4 `poll_vote`: emitir o voto no engine com `DecryptPollVote` (pendente de confirmação, ver design); verificar com teste de contrato e voto simulado persistindo `PollResult`
- [ ] 2.4a Pendente após `poll_vote`: reintroduzir `history.sync` somente quando o business tiver um construtor e fluxo de disparo; o engine não aceita o comando até lá.
- [x] 2.5 Handshake `engine.ready` → business envia `session.start` por conexão ativa; remover `session_loader.go`, o `Sleep(5s)` e o filtro `engineType`; verificar que o engine reinicia sessões sem nenhuma consulta a `Whatsapps`
- [x] 2.6 Mapa de clientes do engine indexado por `(tenantId, connectionId)`; verificar com teste que a mesma id em duas empresas não colide

## 3. Esquema greenfield

- [x] 3.1 Baseline de esquema versionado com ferramenta de migração, substituindo `AutoMigrate` e os DDLs ad-hoc de `database.go`; job de migração com advisory lock; verificar que um banco vazio sobe ao esquema completo e a segunda execução não altera nada
- [x] 3.2 `ChannelConnections` como única tabela; `whatsapp_connection_details` 1:1 para proxy, `keepAlive`, sync, `wid` e risco; verificar com teste de repositório que uma conexão WhatsApp grava e lê seus detalhes
- [x] 3.3 `Tickets.connectionId NOT NULL` com FK; remover `whatsappId` e o sentinela `0`; verificar com teste que ticket de qualquer canal grava e consulta por `connectionId`
- [x] 3.4 Remover `whatsappId` de `Users`, `Flows`, `Assistants`, `CallLogs`, `group_caches`, `group_campaigns`, `group_campaign_targets`, `group_campaign_sends` (um PR por tabela ou grupo coerente); verificar que a busca por `whatsappId` no código de produção retorna zero
- [x] 3.5 `Messages` com chave primária interna e `UNIQUE (tenantId, connectionId, externalId)`; `ON CONFLICT` usa essa chave; verificar com teste de colisão entre conexões e de reentrega
- [x] 3.6 Contato por identidade: telefone opcional, `ContactIdentity` como fonte; verificar com teste que dois contatos do Telegram sem telefone coexistem
- [x] 3.7 Remover `Whatsapps`, `whatsapp_queues` e a rota `/whatsapp/*`; verificar que nenhum SQL ou rota de produção os referencia

## 4. Núcleo de canais

- [x] 4.1 `Channel.ResolveAddress` no contrato e implementação nos adaptadores; verificar que `contactJID` deixa de ser usado no caminho de envio
- [x] 4.2 `MessageDispatchService` com `ReplyPolicy` e `Capabilities`; verificar com teste que excesso de texto e mídia não suportada são recusados com motivo
- [x] 4.3 `message_send.go` fino chamando o serviço; remover a consulta prematura a conexão; verificar que responder ticket do Telegram envia e persiste
- [x] 4.4 Respostas rápidas, reações e mídia usam o serviço; verificar com teste por tipo de mensagem
- [x] 4.5 Adaptador WhatsApp real (conectar, desconectar, enviar com citação, menção e mime, status); verificar com a suíte de contrato e teste de paridade dos cenários de envio atuais
- [x] 4.6 Remover os ramos `EngineType != "whatsmeow"` e resíduos do izapia; verificar com busca que a string não existe
- [x] 4.7 Registro único de canais: eliminar `flow.ChannelRegistry` e `OutboundChannelAdapter`; o FlowBuilder usa o serviço de envio; verificar que um fluxo disparado por ticket do Telegram responde pelo Telegram
- [x] 4.8 Assistentes, campanhas de grupo e plugins usam o serviço; verificar com teste por consumidor
- [x] 4.9 Telegram completo: mídia de saída (foto, vídeo, áudio, voz, documento), citação, divisão por 4096, tratamento de 403 e 429 com `LastError`; verificar com a suíte de contrato estendida
- [x] 4.10 Polling do Telegram com leader lock e retomada no boot; verificar com teste que duas instâncias simuladas não consomem o mesmo token
- [x] 4.11 Canal `fake` e teste de ponta a ponta sem alterar o núcleo; verificar que o teste passa e que nenhum arquivo fora do pacote do canal muda
- [x] 4.12 Edição de conexão (`PUT`) completa com `LastError` e modo webhook/polling; verificar com teste de controller e de interface

## 5. Business em camadas

- [x] 5.1 Lint de arquitetura (`depguard`/`forbidigo`): proíbe `gorm` em `controllers/`, `"Whatsapps"`/`whatsappId`, `@s.whatsapp.net` e importação de adaptadores pelo núcleo; verificar que o CI falha ao violar cada regra
- [x] 5.2 Teste de arquitetura: o núcleo não importa nenhum adaptador concreto; verificar adicionando um import e vendo falhar
- [x] 5.3 Migrar `message`, `ticket`, `contact` e `quick_answer` para controllers finos e repositórios por empresa; verificar que os testes existentes continuam verdes e o lint 5.1 passa neles
- [x] 5.4 Barramento de eventos durável (outbox) no lugar do `go func` por handler; verificar com teste que um evento não se perde se o processo cai entre a gravação e o tratamento
- [x] 5.5 Quebrar arquivos acima de 500 linhas nos pacotes de canal e mensagem; verificar com checagem de tamanho no CI
- [ ] 5.6 Migrar os demais controllers por módulo, um PR por vez; verificar com a contagem de controllers que usam GORM direto caindo até zero

## 6. VoIP

- [x] 6.1 Hooks do `CallManager` assíncronos com fila ordenada por chamada; verificar com teste de ordem e com `-race`
- [x] 6.2 `Capabilities.voice` e `video`; comandos `call.*` com `connectionId`; remover a leitura de `Ticket.WhatsappID` das chamadas; verificar que chamada recebida e iniciada continuam passando nos testes de ponta a ponta
- [x] 6.3 Vídeo atrás de flag experimental, desligada por padrão; atualizar o ADR 0032 com o que existe e o que não foi validado; verificar que com a flag desligada nenhum caminho de vídeo é exposto
- [x] 6.4 Testes caóticos de chamada: UDP bloqueado, perda de relay e reinício do engine no meio da chamada; verificar que a chamada termina com o motivo correto e libera a conexão
- [x] 6.5 Autenticação por token na API de áudio do engine, aceitando só o business; verificar com teste que requisição sem token é recusada
- [x] 6.6 Levantar a procedência de `mic_clip.raw` e das capturas em `testdata` e registrar a decisão do dono; verificar com nota no `NOTICE.md`

## 7. Frontend agnóstico

- [x] 7.1 `useConnections` único e `/connections/:id`; remover `useWhatsApps` e `/whatsapp/*`; verificar com `typecheck` e testes que nenhuma tela usa os antigos
- [x] 7.2 Remover JID do cliente: menções e citações por `contactId` resolvidas no backend; verificar com busca que `@s.whatsapp.net` não existe no frontend
- [ ] 7.3 Ticket serializa `channel` e `capabilities`; lista, cabeçalho e composer reagem à capacidade; verificar com testes de componente por canal
- [ ] 7.4 Edição e exclusão de conexão na interface unificada, com erro de canal visível; verificar com teste de componente e validação visual
- [ ] 7.5 Quebrar componentes acima de 500 linhas e reduzir `any` e `eslint-disable`; verificar com `typecheck`, `lint` e `vitest` verdes
- [ ] 7.6 Testes de componente nos fluxos de conexão, envio e chamada; verificar que a suíte cobre cada fluxo

## 8. Operação

- [x] 8.1 `/metrics` Prometheus nos dois serviços (mensagens, latência, erros por canal, filas, chamadas ativas); verificar que o endpoint expõe cada série
- [x] 8.2 Readiness real (banco, broker, cache) e `trace_id` do webhook ao SSE; verificar que o serviço fica "não pronto" com o broker fora
- [x] 8.3 Piso de cobertura nos pacotes críticos (canais, envio, contrato, autorização) no CI; verificar que o CI falha abaixo do piso
- [x] 8.4 `LastError` visível na interface para todo erro de canal; verificar com teste de ponta a ponta de token revogado
- [x] 8.5 Documentar instância única como premissa do engine, do poller e do `Migrate`; verificar com ADR revisado
- [x] 8.6 Backup, restore e runbooks testados; verificar restaurando em banco limpo

## 9. Governança

- [x] 9.1 ADR 0034 (conexão única, identidade de mensagem, contrato do engine) e ADR do canal `fake`; verificar com revisão
- [x] 9.2 Reconciliar `openspec/changes/add-channel-*` com o código (reabrir as tarefas 4.1 e 4.2 marcadas sem estarem feitas); verificar que cada tarefa marcada corresponde a comportamento existente
- [x] 9.3 Limpar `ESTADO_ORQUESTRATOR.md` e `docs/legacy-*`; corrigir o comentário falso de `session.go`; verificar com revisão
- [x] 9.4 Checklist "novo canal" executável como teste; verificar que o canal `fake` o satisfaz
- [x] 9.5 Portão 1 completo: `go build`, `go vet`, `go test ./... -short -race`, lint v2.12.2, frontend `typecheck`/`lint`/`vitest` e stack local `lt` subindo e atendendo; registrar as evidências no PR

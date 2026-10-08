# Tasks

> Premissas das Open Questions do design (não bloqueiam): alertas de qualidade em RTT > 400 ms, perda > 5% e
> jitter > 60 ms (configuráveis); MP3 mono a 32 kbps; toque com o som já existente
> `frontend/src/assets/sound.mp3`.
> Convenções do repositório: `auth.GetScoped`, `Session(NewDB:true)` por consulta, testes contra Postgres real,
> swagger regenerado em toda rota nova, arquivos < ~250 linhas, `lucide-react` e `src/components/ui`.
> **Divisão**: engine = sinalização, mídia e medição (adaptador burro); business = regra, registro, gravação, S3.

## 1. Pré-requisito: isolamento do SSE (independente das chamadas)

- [x] 1.1 Escrever teste de regressão que reproduz o vazamento: usuário da empresa A abre `?rooms=tenant:<B>` e recebe evento da B; confirmar que o teste **falha** no código atual
- [x] 1.2 Validar `?rooms=` em `controllers/sse.go`: aceitar só `chat:<ticketId>` de ticket visível ao usuário, `tickets:<status>`, `helpdesk-kanban` e `notification`; descartar `tenant:*`, `user:*` e qualquer outra; verificar que o teste 1.1 passa e que os testes de SSE existentes continuam verdes
- [x] 1.3 Inscrever no servidor a sala `user:<tenantId>:<userId>` a partir do token (nunca do query) e testar que dois usuários da mesma empresa não recebem eventos um do outro
- [x] 1.4 Adicionar `SSEHub.HasSubscribers(room)` e testar com conexões abertas, fechadas e salas inexistentes

## 2. Engine: porte da pilha VoIP

- [x] 2.1 Copiar `core`, `wanode`, `signaling`, `transport`, `call`, `media` e `media/mlow` do WaCalls (commit registrado) para `engine-go/internal/voip/`, ajustando só os imports; manter `LICENSE` (MIT jotadev66) e `mlow/LICENSE` (MIT Rajeh Taher) e criar `NOTICE.md` com origem e commit; verificar `go build ./...` no Go 1.25
- [x] 2.2 Trazer testes e `testdata` do `mlow` e rodar `go test ./internal/voip/...` no engine; verificar que passam (referência: passam no clone do WaCalls em Go 1.25)
- [x] 2.3 Adicionar `pion/webrtc/v4` ao `go.mod` do engine e confirmar `CGO_ENABLED=0` e que a imagem `distroless` builda
- [x] 2.4 Implementar o adaptador `VoipSocket` sobre o `*whatsmeow.Client` do engine e testar com `whatsmeow` fake que `SendNode`/`Query` respeitam timeout e cancelamento
- [x] 2.5 Enviar `offer`/`accept` em goroutine, fora do laço serial de eventos; verificar com teste que uma mensagem é processada enquanto um `offer` aguarda ack por 15 s
- [x] 2.6 Isolar cada chamada com `recover`: um pânico no codec ou no transporte encerra só aquela chamada e publica `call.ended` com `failed`; verificar com teste que injeta pânico

## 3. Engine: ciclo de vida, comandos e eventos

- [x] 3.1 Criar o registro `callID → CallManager` por sessão com no máximo **uma** chamada ativa por conexão; verificar que a 2ª oferta **não é atendida nem recusada** e gera `call.missed` com motivo `busy`
- [x] 3.2 Registrar handlers de `CallOffer`, `CallAccept`, `CallTerminate`, `CallTransport`, `CallRelayLatency`, `CallOfferNotice` e `UnknownCallEvent` sem bloquear o laço de eventos; verificar com teste que cada evento chega ao gerenciador certo
- [x] 3.3 O engine **nunca** envia `reject` por conta própria: sem operador elegível, tipo não suportado (vídeo/grupo), proxy configurado ou conexão ocupada, ele **ignora** a oferta e publica `call.missed` com o motivo; verificar com teste que nenhum nó `reject`/`terminate` é enviado nesses casos
- [x] 3.4 Enviar `preaccept` somente após o comando `call.ready` do business (ver 6.3); sem confirmação em ~3 s, abandonar a oferta e publicar `call.missed`; verificar com relógio simulado
- [x] 3.5 Ignorar e não abrir mídia em sessão com proxy (`proxyUrl` não vazio); verificar com teste que nenhum socket UDP é aberto e que o evento traz `proxy_blocked`
- [x] 3.6 Resolver o telefone do chamador (`CallCreator`/`CallCreatorAlt` + store LID↔PN, reaproveitando `resolveChatPN`) e enviar `callerPn`; verificar os cenários com e sem mapa
- [x] 3.7 Criar a fila `engine.go.calls` ligada a `wbot.*.*.call.*` em `wbot.commands`, com **consumidor próprio** (canal e goroutine separados); remover `call.*` do binding de `engine.go.commands`; verificar com teste que um comando `call.accept` é processado enquanto o consumidor de mensagens está bloqueado em um envio lento
- [x] 3.8 Implementar os comandos `call.ready`, `call.accept`, `call.reject`, `call.end` e `call.start`, cada um despachado em goroutine própria; verificar cada um com teste de contrato do payload e que `call.reject` só ocorre por comando
- [x] 3.9 Publicar `call.incoming`, `call.state`, `call.ended`, `call.missed` e `call.quality` em `wbot.events` com `callId`, `peer`, `callerPn`, `direction`, `endReason`; verificar o payload com teste de contrato
- [x] 3.10 Timeout de toque de 45 s para ofertas e chamadas originadas; verificar com relógio simulado que encerra como `timeout`
- [x] 3.11 Tratar "atendida em outro aparelho": ao receber aceite de outro dispositivo, encerrar e publicar `accepted_elsewhere`; verificar com teste de sinalização
- [x] 3.12 Chamada órfã após reinício: o engine guarda chamadas só em memória, então ao (re)iniciar uma sessão ele publica `call.reset`; o business marca como `interrupted` toda chamada ativa daquela conexão (ver 6.8). Em parada/queda da sessão com chamada em curso o engine abandona localmente e publica `call.ended` com `interrupted`; verificar com teste

## 4. Engine: canal de áudio e medição

- [x] 4.1 Endpoint interno `GET /calls/{id}/audio` (WebSocket) autenticado por `X-Internal-Token` com comparação em tempo constante, só `expose`, **não sobe sem `CALLS_AUDIO_TOKEN`**; verificar que sem token o servidor não sobe e token errado dá 401
- [x] 4.2 Ligar o WebSocket a `FeedCapturedPCM` e `OnPeerAudio` (quadros de 640 bytes, 16 kHz Int16 LE) com fila limitada e descarte do excedente; verificar que o engine não bloqueia com consumidor lento e que a fila não cresce sem limite
- [x] 4.3 Medir perda e jitter do áudio recebido (lacunas de sequência RTP; jitter RFC 3550 sobre o timestamp) e verificar com sequências sintéticas (sem perda, 10% de perda, reordenação, jitter conhecido)
- [x] 4.4 RTT do relay: usa só o `c2r_rtt` da oferta (menor entre os relays; `rttMs` nulo se ausente), testado com relés simulados. O par `ping`→`pong` NÃO é usado: o `ping` tem `txid` de 12 bytes e o parser lê o do `pong`, mas não foi validado contra o relay real que ele devolve o mesmo `txid`, e o `onRelayData` descarta STUN. Validar em chamada real (10.4) antes de ligar
- [x] 4.5 Medir bytes/s TX e RX, RMS do PCM nos dois sentidos, estado e reconexões do relay e "sem áudio do contato há > 5 s"; verificar cada medida com entrada sintética
- [x] 4.6 Emitir a telemetria a 1 Hz como mensagem de controle (JSON texto) no WebSocket e no evento `call.quality`; verificar o formato com teste de contrato
- [x] 4.7 Liberar todos os recursos (relay, SRTP, goroutines, canal) em qualquer término; verificar com teste de vazamento de goroutines em encerramento normal, falha e queda do WebSocket
- [x] 4.8 Expor carga no `/health` (chamadas ativas, tamanho das filas de áudio); verificar com teste do endpoint

## 5. Business: dados e permissões

- [x] 5.1 Criar model e migração de `CallLogs` (`tenantId`, `whatsappId`, `contactId`, `ticketId` nulo, `callId`, `direction`, `status`, `startedAt`, `answeredAt`, `endedAt`, `durationSec`, `handledByUserId`, `endReason`, resumo de qualidade `rttAvg/rttMax/lossAvg/lossMax/jitterAvg/jitterMax/mosEstimated`, e `recordingKey/recordingStatus/recordingDurationSec`) com índice único `(tenantId, callId)` e índice `(tenantId, startedAt)`; verificar contra Postgres real que a duplicata no mesmo tenant é rejeitada e em outro é aceita
- [x] 5.2 Criar `CallRecordingAccess` (`callId`, `userId`, `tenantId`, `action`, `at`); verificar migração e isolamento por tenant
- [x] 5.3 Adicionar `calls:receive`, `calls:place`, `calls:read`, `calls:delete` e `calls:manage` ao catálogo com descrições em pt e **não** anexá-las a nenhum cargo do seed; verificar que cargos do seed não as têm e que `RequirePermission` as exige
- [x] 5.4 Setting por empresa `callRecordingMode` (`off` padrão, `optional`, `auto`) com `callRecordingAckBy/At`: ausente ou valor inválido equivale a `off`; as três chaves **não** mudam pelo `PUT /settings/:key` genérico (devolve 403) porque `settings:update` não cobre o aceite. A rota própria com aceite é a 7.12
- [x] 5.5 Mascarar/validar o valor da setting para quem não tem `calls:manage`; verificar com teste de leitura por usuário comum

## 6. Business: eventos, elegibilidade e registro

- [x] 6.1 Consumir `call.incoming`, `call.state`, `call.ended`, `call.missed`, `call.quality` e `call.reset` em **fila própria** `api.events.calls.go` (não em `api.events.process.go`, para o toque não esperar atrás das mensagens), **idempotente** por `(tenantId, callId)`; verificado que a mesma entrega duas vezes gera um único `CallLogs`, um único `call.ready` e um único toque
- [x] 6.2 Resolver contato e ticket (`FindOrCreate(..., knownNumber)`); verificar que oferta por LID cai no contato da agenda sem duplicar e que sem telefone cria contato por LID
- [x] 6.3 Calcular elegíveis: online (`HasSubscribers`), com `calls:receive` (ou alcance de empresa), **alcance sobre a conexão** (conexão do usuário ou ligada a fila do usuário) e **não pausados**; se houver, enviar `call.ready` e emitir `call.incoming` na sala de cada um; se não, não enviar `call.ready` e registrar `missed`; verificar com usuários elegíveis, sem permissão, de outra empresa, fora do alcance e pausados
- [x] 6.4 Consulta reutilizável "usuários que enxergam a conexão X" espelhando `GetScopedDB`; **teste de paridade** com a visibilidade de tickets. O teste achou uma divergência do desenho original: `User.WhatsappID` NÃO dá visibilidade de ticket (é só a conexão preferida ao criar), então ficou de fora
- [x] 6.5 Atribuição atômica (`UPDATE ... WHERE status='ringing' AND handledByUserId IS NULL` + `RowsAffected`); verificar com dois atendimentos concorrentes que só um vence e o outro recebe "já atendida"
- [x] 6.6 Ao encerrar, finalizar `CallLogs` com duração real e resumo de qualidade e gravar `Message` de sistema (`mediaType: "call"`, `dataJson`) no ticket, emitindo `appMessage`; verificar o histórico do ticket
- [x] 6.7 Chamada perdida cria ticket `pending` se não houver ticket aberto na conexão, ou usa o existente; verificar os dois casos
- [x] 6.8 Chamada interrompida: ao receber `call.reset` (engine reiniciou a sessão) ou `call.ended` com `interrupted`, marcar `interrupted` toda chamada ativa daquela conexão e remover o toque; verificar com teste
- [x] 6.9 Interpretar `call.quality`: índice estimado (modelo E simplificado), nível de 3 estágios e alerta por limites; verificar o cálculo com tabela de entradas e saídas conhecidas e os limiares de borda

## 7. Business: rotas, áudio e gravação

- [x] 7.1 Rotas `POST /calls` (ligar), `POST /calls/:id/accept`, `/reject`, `/end`, `GET /calls` (histórico), `GET /calls/:id` com `RequirePermission` correto e `auth.GetScoped`; verificar 403/401/404, escopo de empresa e regenerar o swagger
- [x] 7.2 `POST /calls` valida ticket individual, conexão conectada e **sem proxy**, `calls:place` e que o operador não está em outra chamada; verificar cada negativa contra Postgres real
- [x] 7.3 Histórico restrito à empresa e, sem alcance de empresa, aos tickets visíveis; verificar isolamento entre empresas e entre operadores
- [x] 7.4 Adicionar `coder/websocket` e implementar `GET /calls/:id/audio`: valida token, empresa e operador que assumiu; proxy de bytes ao WebSocket do engine; verificar que usuário de outra empresa ou outro operador é rejeitado e que os bytes chegam íntegros
- [x] 7.5 Encerrar a chamada se o WebSocket do navegador ficar caído por mais de 10 s; verificar com tempo simulado
- [x] 7.6 Backpressure no proxy de áudio (fila limitada, descarte do excedente, nunca bloqueia); verificar com consumidor lento
- [x] 7.7 Vendorizar o `shine-mp3` em `business/internal/recording/shine` com `LICENSE` (LGPL-2.0) e `NOTICE.md` e a **única** alteração de aceitar o bitrate como parâmetro; verificar `go build` e que o diff contra o original é só essa alteração
- [x] 7.8 Gravador: mixador mono 16 kHz que soma operador e contato por relógio, com lacunas viradas silêncio e limitação de pico, acumulando em **múltiplos de 576 amostras** antes de codificar e completando o último quadro no fechamento; verificar com teste de ida e volta usando um **decodificador independente** (`go-mp3`): duração dentro de 1 s, 16 kHz, e que blocos de 20 ms **não** corrompem (regressão da armadilha medida)
- [x] 7.9 Gravar em arquivo temporário e, ao fim, enviar ao `ObjectStore` com chave `{tenantId}/calls/{callId}.mp3`, guardando só a chave; em falha de upload, registrar `recordingStatus=failed` sem expor áudio parcial; verificar com `ObjectStore` falso (sucesso, falha, S3 ausente)
- [x] 7.10 Rotas de gravação: `POST /calls/:id/recording/start|stop` (modo `optional`), `GET /calls/:id/recording` (URL assinada + registro em `CallRecordingAccess`) e `DELETE /calls/:id/recording` (`calls:delete`, remove do S3 e registra); verificar permissão, escopo, auditoria e que fora do alcance devolve 404 sem revelar existência
- [x] 7.11 Modo `auto` inicia a gravação ao conectar o áudio; modo `off` rejeita iniciar; instalação sem S3 rejeita e oculta; verificar os três modos
- [x] 7.12 Rota de configuração da gravação com aceite de responsabilidade (`calls:manage`); verificar que sem aceite não muda e que o aceite grava usuário e horário

## 8. Frontend

- [x] 8.1 Módulo de captura/reprodução com `AudioWorklet` (16 kHz mono Int16, quadros de 20 ms, jitter buffer ~60 ms com descarte acima de ~200 ms) e hook de WebSocket de áudio; verificar com testes unitários do reamostrador e do buffer
- [x] 8.2 Provedor global de chamadas no `MainLayout`: assina `call.*` na sala do usuário e mantém o estado; verificar que o toque some quando outro atende ou o chamador desiste
- [x] 8.3 Modal de chamada recebida (nome, conexão, Atender/Recusar, som em loop) exibido em **qualquer tela** só com `calls:receive`; verificar com teste de renderização com e sem permissão
- [x] 8.4 Tela de chamada ativa (cronômetro desde a conexão real, silenciar, encerrar, "Chamando…", gravar, indicador "gravando") e tratamento de microfone negado; verificar cada estado
- [x] 8.5 Painel de qualidade: indicador de 3 níveis, detalhes a 1 Hz (RTT, perda, jitter, taxa, nível TX/RX), alerta visual, rótulo "estimado" e nota do limite de medição; verificar com telemetria simulada em cada nível
- [x] 8.6 Botão "Pausar chamadas" no cabeçalho com preferência local e aviso de presença ao servidor; verificar que pausado não conta como elegível
- [x] 8.7 Botão de ligar no `TicketHeader`/`TicketActionButtons` (só ticket individual, com `calls:place`; desabilitado com motivo se desconectado ou com proxy); verificar com testes de renderização
- [x] 8.8 Renderizar a mensagem `mediaType: "call"` no histórico (direção, operador, duração, resultado, player da gravação quando houver); verificar com teste de componente
- [x] 8.9 Menu **Chamadas** (`SidebarNav`, rota `/calls`, `calls:read`) com tabela, filtros, coluna de qualidade, player e exclusão (`calls:delete`); verificar visibilidade por permissão
- [x] 8.10 Seção "Chamadas" em Configurações (`calls:manage`): modo de gravação com texto de responsabilidade e confirmação; verificar que a mudança só ocorre após o aceite e que usuário sem `calls:manage` não vê a seção
- [x] 8.11 Traduções `pt`, `en` e `es` de todos os textos novos; verificar com teste que as três línguas têm as mesmas chaves
- [x] 8.12 Usar `lucide-react` e `src/components/ui`, sem MUI nem emoji estrutural; verificar com `npm run lint`, `npm run typecheck` e `npx vitest run` sem avisos

## 9. Documentação

- [x] 9.1 `docs/user/calls/USING_CALLS.md`: guia do operador (receber, ligar, qualidade, pausar, limitações); verificar que cobre todos os cenários do spec de interface
- [x] 9.2 `docs/user/calls/ENABLING_CALLS.md`: guia do administrador (conceder permissões em Acessos → Cargos, quem passa automaticamente, configurar e aceitar a gravação, requisito de S3); verificar passo a passo contra a tela real
- [x] 9.3 `docs/user/plugins/MARKETPLACE_AND_MENU.md`: como o Marketplace e o menu lateral funcionam (onde fica, `free` × `pro`, abas, o que o Ativar faz, regra `activePlugins` do menu, e por que um plugin `free` novo não aparece sem entrada no Hub); verificar os caminhos e rotas contra o código
- [x] 9.4 ADR novo (chamadas de voz: porte do WaCalls, divisão engine × business, fila dedicada, WebSocket PCM, gravação no business, bloqueio com proxy, correção do SSE) e `docs/agents/calls.md` com invariantes; atualizar a seção de módulos do `CLAUDE.md`; verificar links e ausência de segredos
- [x] 9.5 Documentar a saída UDP necessária do engine e a nota de atualização (alcance de empresa passa a poder receber toque; gravação nasce desligada); verificar que a mensagem de erro de UDP bloqueado é clara

## 10. Infra e verificação final

- [x] 10.1 Variáveis `CALLS_AUDIO_URL`/`CALLS_AUDIO_TOKEN` e `expose` (nunca `ports`) no `docker-compose.*` do engine; verificar que o compose valida e que o token é obrigatório **Revisado em 2026-10-06 (ADR 0031):** `CALLS_AUDIO_URL` e `CALLS_AUDIO_TOKEN` foram removidos; o business usa só `ENGINE_HOST`.
- [x] 10.2 Verificação completa (2026-10-06, Postgres e RabbitMQ reais, sem avisos). **Engine**: `go vet` limpo; `go test ./... -race` = **314 passam, 2 skips** do `mlow` herdados do WaCalls (12 pacotes); os 3 testes de fila só rodam com `AMQP_TEST_URL` e passaram. Build `CGO_ENABLED=0` estático. **Business**: `go vet` limpo, gofmt limpo nos arquivos alterados; **1338 passam, 0 falhas** (outros pacotes 511 + controllers 373 + services 154 + database 10 + plugins 201 + repository 89; 2 skips de teste que exigem variável de ambiente). **Frontend**: `typecheck` e `lint --max-warnings 0` limpos; `vitest` = **433 passam em 53 arquivos** (eram 204 em 31 antes desta mudança). Nenhum teste existente foi enfraquecido para passar
- [x] 10.3 Teste de integração com RabbitMQ e Postgres reais: oferta fictícia → elegibilidade → atender → áudio sintético → telemetria → encerrar → `CallLogs` e gravação (`business/internal/calls/e2e_flow_test.go`, `AMQP_TEST_URL`). O lado do engine é simulado por um cliente AMQP e um WebSocket reais que falam o contrato exato; verificado sem WhatsApp, limpo com `-race`, e provado desfazendo três elos (call.ready, telemetria, gravação). Mais um teste de contrato do JSON de comando entre business e engine
- [x] 10.4 Roteiro de **teste manual com dois números reais** escrito em `manual-test-script.md` (desta pasta; ao arquivar a change saiu de `openspec/changes/add-whatsapp-voice-calls/`): pré-requisitos, 47 passos com critério de aceite e campo de resultado (receber, atender, recusar, perder, atender em outro aparelho, fazer chamada, proxy, permissões, falhas de rede/áudio/UDP, gravação nos três modos, carga e isolamento) e as 6 perguntas que só a chamada real responde. Conferido contra o spec: 74 cenários, dos quais 46 exercitados à mão e 28 só automatizados (não dependem de WhatsApp real). **Não foi preenchido item a item** (a validação ao vivo foi feita pelo dono em chamadas reais; ver 10.5)
- [x] 10.5 Imagens `:test` do engine e do business publicadas e **validadas ao vivo pelo dono** em chamadas reais com dois números (2026-10-06 a 2026-10-08), com 8 defeitos achados e corrigidos no caminho (ver 10.6 e "Cobertura ao vivo" em `manual-test-script.md`). **O roteiro de 47 passos não foi preenchido item a item**: os passos sem prova ao vivo e as 6 perguntas abertas (qualidade do áudio, RTT, ban após N chamadas, NAT estrito) ficam registrados como não validados, não como aprovados
- [x] 10.6 (extra, feito antes de 10.5) Correções trazidas de forks do WaCalls antes de publicar (2026-10-06; procedência e commits em `engine-go/internal/voip/NOTICE.md`). **SRTP**: `Unprotect` agora verifica a tag HMAC e tem janela anti-replay de 64 pacotes (RFC 3711); antes pacote forjado, adulterado ou repetido era aceito (provado por teste). Isso também elimina a duplicação de áudio quando o mesmo pacote chega por vários relays (3 relays davam 150 decodificações para 50 pacotes; agora 50, `callmanager_dedup_test.go`). **Codec mlow**: tabela de twiddle da FFT e buffers reutilizados; codificar 60 ms caiu de ~10,0 para ~3,9 ms (cerca de 6 para 15 chamadas por núcleo) e os 160 pacotes de referência saem **idênticos byte a byte** ao porte original. **Sinalização**: `reject`/`terminate` deixam de ser abandonados quando o contexto do comando já foi cancelado, e `terminate` vai ao aparelho que atendeu. Cada correção tem teste que falha sem ela (mutação). Engine: 335 testes passam com `-race`. **Não trazido**: estado "reconectando" com redial de relay (~960 linhas, não validável sem chamada real), alterações de capabilities (os forks se contradizem), vídeo e grupo (mudam o escopo)

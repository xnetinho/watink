# Chamadas de voz do WhatsApp — Contexto para Agentes

> **Status:** implementado (ADR 0031). **Não validado com chamada real**: ver §Validação.

## Responsabilidade
Receber e fazer chamadas de voz **1:1** pelo navegador, com qualidade ao vivo, histórico no ticket,
gravação opcional em MP3 e auditoria. Recurso **nativo do core** (sem plugin, sem Marketplace).

## Arquitetura
```
Navegador ──WSS PCM──► business ──WS interno──► engine ──UDP──► relays do WhatsApp
   ▲  SSE (sala user:<tenant>:<id>)   │ AMQP: wbot.<t>.<s>.call.*   │ whatsmeow (sinalização)
   └───────────────────────────────── ┴─────────────────────────────┘
```
- **engine** `engine-go/internal/{voip,calls,callsapi}`: sinalização, mídia, medição. Adaptador burro.
- **business** `business/internal/{calls,recording}` + `controllers/call*.go`: regras, registro, S3.
- **frontend** `src/{context/Calls,components/Calls,lib/calls,pages/Calls}` + seção em Settings.

## Invariantes
**Segurança e isolamento**
- `?rooms=` do SSE passa por lista permitida; `tenant:*` e `user:*` **nunca** vêm do query. A sala
  pessoal é inscrita pelo servidor a partir do token (`controllers/sse_rooms.go`).
- Toda query em `CallLogs`/`CallRecordingAccess` carrega `WHERE "tenantId"` manual (RLS é inerte no worker).
- Quem **não pode ver** um ticket/gravação recebe **404**, nunca 403 (não revela que existe).
- Telemetria (`call.quality`) vai **só** ao operador que assumiu a chamada, nunca à sala da empresa.
- WebSocket do navegador: token na query (como o SSE), `calls:receive` **ou** `calls:place`, chamada
  da própria empresa e **operador que a assumiu**; um canal por chamada; `Origin` restrito.
- Canal de áudio do engine: **sem autenticação própria** (decisão do dono: rede interna, como o RabbitMQ), **só `expose`**, nunca `ports:` — quem alcança a porta alcança o áudio.

**Regras de chamada**
- O engine **nunca** envia `reject`/`terminate` por conta própria; só por comando de operador.
  `preaccept` só depois de `call.ready` do business.
- Sem operador elegível, com proxy, de vídeo/grupo ou conexão ocupada: **ignora** e publica `call.missed`.
- Elegível = enxerga a conexão (paridade com `GetScopedDB("Tickets")`, **sem** `User.WhatsappID`) +
  `calls:receive` + online (`SSEHub.HasSubscribers`) + não pausado.
- Atribuição atômica (`UPDATE ... WHERE status='ringing' AND handledByUserId IS NULL` + `RowsAffected`).
- Eventos idempotentes por `(tenantId, callId)`; a mensagem do ticket tem id `call:<callId>`.
- Uma chamada ativa por conexão e por operador; nenhum teto por empresa.
- Conexão com proxy (`proxyMode`, `proxyId` **ou** `proxyGroupId`) → sem chamada, **fail-closed**.
- Prazos: 3 s (`call.ready`), 45 s de toque, 25 s para a mídia conectar, 10 s sem o canal do navegador. O toque de 45 s é cancelado ao atender (`trackMedia`), nos dois sentidos.
- `answeredAt`: gravado em `Accept` (entrada) ou no primeiro `call.state active` (saída); separa `ended` de `missed` em `statusForEnd`. `<reject>` do contato encerra como `declined`, `<terminate>` como `user_ended`.

**Áudio e gravação**
- Formato fixo: PCM 16 kHz mono Int16 LE, quadros de **640 B**. Filas **limitadas** com descarte do mais antigo.
- O encoder MP3 só recebe **blocos de 576 amostras**; o último é completado com silêncio.
- Mixagem por relógio de 20 ms: lacuna vira silêncio, pico é limitado (nunca dá a volta).
- Banco guarda a **chave** do objeto, nunca URL assinada; a URL (5 min) é gerada a cada leitura.
- `callRecordingMode`: ausente/inválido = `off`; sair de `off` exige `ack=true`; as chaves **não** mudam pelo
  `PUT /settings/:key` genérico. Sem S3, só `off`.
- Escuta e exclusão gravam `CallRecordingAccess` **antes** do ato; sem expiração automática.

## O que NÃO fazer
- Não enviar `reject` pelo engine. Não decidir regra de negócio no engine.
- Não usar `User.WhatsappID` como visibilidade de conexão.
- Não alimentar o `shine-mp3` com blocos que não sejam múltiplos de 576 amostras.
- Não aceitar sala de SSE vinda do query sem passar por `allowedExtraRooms`.
- Não expor o canal do engine em `ports:`. Não confundir a porta 8085 com uma API autenticada.
- Não tratar `media_timeout` como erro do usuário: costuma ser saída UDP bloqueada.
- Não criar uma segunda sessão do WhatsApp para chamadas (o aparelho vinculado é um só).
- Não confundir com `Campaign`/`CampaignRecipient` (FlowBuilder) nem com plugin: não há `PluginInstallations` aqui.

## Eventos AMQP (`wbot.<tenant>.<sessionId>.<tipo>`)
- **Comandos** (`engine.go.calls`): `call.ready`, `call.accept`, `call.reject`, `call.end`, `call.start`.
- **Eventos** (`api.events.calls.go`): `call.incoming`, `call.state`, `call.ended`, `call.missed`,
  `call.quality`, `call.reset`. Motivos de fim: `user_ended`, `declined`, `timeout`, `busy`, `cancelled`,
  `failed`, `no_operator`, `proxy_blocked`, `unsupported_type`, `accepted_elsewhere`, `interrupted`, `media_timeout`.

## Rotas
`POST /calls` (`place`) · `GET /calls`, `GET /calls/:id` (`read`) · `POST /calls/:id/accept|reject` (`receive`) ·
`POST /calls/:id/end` (`receive` ou `place`) · `PUT /calls/pause` (`receive`) ·
`GET /calls/:id/audio` (WebSocket, token na query) · `POST /calls/:id/recording/start|stop` (`receive`/`place`) ·
`GET|DELETE /calls/:id/recording` (`read` / `delete`) · `GET|PUT /calls/recording-config` (`manage`).

## Variáveis de ambiente
`ENGINE_HOST` (business: o único endereço do engine; dele saem `/health`, grupos e o áudio) ·
`CALLS_AUDIO_PORT` (engine, padrão 8085) · `CALLS_AUDIO_ORIGINS` (business, opcional) ·
`S3_*` (para gravar).

## Validação
Coberto por testes: engine (`-race`, RabbitMQ real), business (Postgres real, WebSockets reais), frontend
(`vitest`). **Não coberto sem WhatsApp real**: sinalização, relay UDP, codec MLow ponta a ponta, qualidade
do áudio, `c2r_rtt`/`ping`→`pong`. Roteiro em `openspec/changes/add-whatsapp-voice-calls/tasks.md` (10.4).

## Referências
ADR 0031 · [`docs/user/calls/`](../user/calls/) · ADRs 0016, 0019, 0021, 0022 ·
`engine-go/internal/voip/NOTICE.md` · `business/internal/recording/shine/NOTICE.md`

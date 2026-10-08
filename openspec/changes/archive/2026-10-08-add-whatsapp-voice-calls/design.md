# Design

## Context

Estado observado no código (não suposições):

**Serviços e fronteiras**
- **engine-go**: único dono da sessão WhatsApp (`whatsmeow`), Go 1.25, imagem `distroless/static`
  (`CGO_ENABLED=0`, sem shell). Fala **só** com RabbitMQ e Postgres; do banco usa as tabelas do `whatsmeow` e um
  `SELECT` em `Whatsapps` ao subir. **Não toca** `Tickets`, `Messages`, `Contacts` nem `Users` (verificado).
  O `CLAUDE.md` o define como "adaptador burro".
- **business**: API REST, regras, RBAC, tickets, SSE, S3. Imagem **Alpine** (não distroless: o `CLAUDE.md` diz
  o contrário, mas o `Dockerfile.business` e a imagem publicada têm `apk`, `wget` e shell). É o **único serviço
  exposto** (Traefik). O frontend React é compilado e **embutido** no binário do business.
- **RabbitMQ** é o único canal engine↔business: comandos em `wbot.commands`, eventos em `wbot.events`, routing
  key `wbot.<tenant>.<session>.<cmd>`. O business **não tem acesso** ao `whatsmeow.Client`.
- O business consome eventos num loop e cada consumidor tem canal próprio (`startConsumer`).

**Pontos que condicionam o desenho**
- O engine consome **todos** os comandos de **todas** as empresas num **único loop sequencial**
  (`for d := range msgs { handler(d) }`, uma Delivery por vez). O time já mitigou à mão um caso real
  (`media.download` em goroutine). É o débito "filas globais" do `CLAUDE.md`.
- O `whatsmeow` do engine (`v0.0.0-20260630`) já despacha `events.CallOffer/Accept/Terminate/Transport/
  RelayLatency` e oferece `RejectCall`, mas o engine **não registra handler de chamada**: ofertas são
  descartadas. Para a sinalização completa expõe `DangerousInternals()`, e todos os métodos que o WaCalls usa
  existem nesta versão (verificado).
- **WaCalls** (MIT) é o único que roda 100% Go sem cgo. `internal/voip` (~3,5 mil linhas de sinalização,
  transporte e chamada + ~11 mil do codec `mlow`, MIT de Rajeh Taher) compila no Go 1.25 e **seus testes
  passam**. O **astracalls** (AGPL) é derivado do WaCalls; seu codec exige cgo e `.so` binário (inviável no
  engine distroless e não auditável): só referência de design.
- O `CallManager` do WaCalls representa **uma** chamada; para várias o servidor cria um por `callID`. A mídia
  usa `pion/webrtc` só como pilha SCTP/DTLS contra os **relays do WhatsApp por UDP direto**, sem proxy.
- Áudio: PCM mono 16 kHz Int16; o MLow opera a 16 kHz em quadros de **960 amostras = 60 ms**.
- **Custo medido** (esta máquina, 4 CPUs): codificar um quadro de 60 ms custa 9,8 ms (≈16% de um núcleo por
  chamada), decodificar 0,19 ms (≈0,3%). Sem teto de chamadas, **CPU do engine é o limite real**.
- **Telemetria disponível** na pilha: `c2r_rtt` na oferta do relay; par `ping`/`pong` do transporte a cada
  1,1 s; cada pacote RTP traz número de sequência e timestamp (perda e jitter RFC 3550). **Não há RTCP**: a
  perda no sentido operador→contato **não é mensurável**.
- Em `controllers/sse.go`, `?rooms=` é aceito **sem validação**. Reproduzido contra o `SSEHub` real: um usuário
  da empresa A que peça `?rooms=tenant:<id-da-B>` recebe os eventos da B.
- `RequirePermission` libera `alcance` `tenant`/`plataforma` **sem consultar o cargo**; `GetScopedDB` filtra
  tickets de alcance comum por `userId`, fila do usuário ou conexão ligada a fila do usuário
  (`user_queues` × `whatsapp_queues`). Essa regra existe como SQL de **ticket**, não como consulta reutilizável
  "quem enxerga esta conexão".
- `domain.ObjectStore` (S3/MinIO, ADR 0019) é **opcional** (`nil` sem `S3_ENDPOINT`/`S3_BUCKET`) e já gera URL
  assinada temporária (`PresignedGetURL`); o padrão de evidência de Atividades guarda só a **chave**.
- Mensagens têm `mediaType` e `dataJson`; o frontend consome `appMessage` via SSE.
- Plugins `free` novos não entram no Marketplace sem entrada no catálogo do Hub (testado contra a imagem do
  `plugin-manager`: em modo stub o catálogo vem vazio e só `helpdesk`/`webchat` são "free" conhecidos). Por isso
  as chamadas são **core nativo**, não plugin.

Ver `proposal.md` (motivação, escopo) e `specs/whatsapp-voice-calls/spec.md` (requisitos).

## Goals / Non-Goals

**Goals:**
- Reaproveitar a pilha VoIP do WaCalls no `engine-go`, sem cgo, **mantendo o engine como adaptador burro**:
  só o que exige a sessão do WhatsApp mora nele.
- Toda regra, permissão, registro, gravação e armazenamento no `business`.
- Áudio do navegador pelo caminho HTTPS já existente (WSS), sem porta UDP nova.
- Comandos de controle de chamada que **não esperam** mensagens de outras empresas.
- Isolamento entre empresas em todos os pontos, inclusive corrigindo o SSE.

**Non-Goals:**
- Vídeo, espera, transferência, chamadas em grupo, transcrição, TTS e IVR.
- Chamadas em conexões com proxy (bloqueadas, Decisão 9).
- Aviso sonoro ao contato, retenção automática e permissão própria para ouvir gravação (decisões do produto,
  registradas na Decisão 7).
- Corrigir o débito global de filas do engine (só isolamos os comandos de chamada).
- Substituir ou "forkar" o `whatsmeow`.

## Decisions

### 1. Divisão engine × business pelo critério "precisa da sessão do WhatsApp?"
| Responsabilidade | Onde | Motivo |
|---|---|---|
| Sinalização `<call>`, cifra da chave | engine | só ele tem o `whatsmeow.Client` |
| Mídia SRTP/SCTP, codec MLow | engine | a chave da chamada vem da sessão cifrada; latência de 60 ms |
| Medição (RTT, sequência RTP, bytes, nível) | engine | só ele vê os pacotes |
| Quem pode atender, elegibilidade, permissões | business | usuários, cargos, tickets |
| `CallLogs`, mensagem no ticket, SSE | business | banco e tempo real |
| **Gravação: mixagem, MP3, S3** | **business** | o PCM já atravessa o business; ele já fala com o S3 |
| Interpretar telemetria, índice, alerta | business | regra de produto, ajustável sem tocar o engine |

- *Alternativa descartada*: engine com API HTTP completa e encoder de MP3 (a v1 deste plano). Mais código e
  superfície no engine, e o encoder LGPL ficaria na imagem do engine. O business **já recebe o PCM** dos dois
  lados, então gravar lá não custa um byte a mais de transporte.

### 2. Porte de `internal/voip` do WaCalls para `engine-go/internal/voip`
O WaCalls é `package main`/módulo `wacalls`, Go 1.26.4. Vendorizamos `core`, `wanode`, `signaling`,
`transport`, `call`, `media` e `media/mlow`, ajustando só o import path, preservando `LICENSE` (MIT, jotadev66)
e `mlow/LICENSE` (MIT, Rajeh Taher) e registrando origem e commit em `engine-go/internal/voip/NOTICE.md`.
Nada do `cmd/server` (broker, sessões SQLite, bridge) é trazido: o engine já tem sessões, banco e mensageria.
- *Alternativas*: importar como módulo (impossível, é `main`); astracalls (cgo); escrever do zero (engenharia
  reversa de protocolo e codec, risco desproporcional); sidecar WaCalls (duplica a sessão do WhatsApp).

### 3. Um `CallManager` por chamada, no máximo uma ativa por conexão
Registro `callID → CallManager` por sessão. A segunda oferta na mesma conexão **não é atendida** e o engine
**não envia recusa**: o celular continua tocando, e o business registra "perdida por ocupação". Sem teto por
empresa: o risco é de cada empresa e o limite natural é a CPU (≈16% de um núcleo por chamada), exposta como
métrica (Decisão 11).

### 4. O engine nunca recusa por conta própria
`RejectCall` envia `reject` ao chamador e o WhatsApp o trata como recusa da **conta inteira**, derrubando o
toque nos demais aparelhos (o changelog do astracalls registra esse efeito). Portanto o engine só **recusa por
comando explícito** de um operador. Sem operador elegível, proxy configurado, tipo não suportado ou conexão já
em chamada, ele **ignora a oferta**: nada é enviado e o celular segue tocando.
- O WaCalls envia `preaccept` logo na oferta. O engine **só envia `preaccept` depois que o business confirma
  que há operador elegível** (evento `call.incoming` → comando `call.ready`), para não sinalizar ao chamador
  "estou atendendo" quando ninguém vai atender. Se a confirmação não chegar em ~3 s, a oferta é abandonada.

### 5. Comandos de chamada em fila dedicada
Nova fila `engine.go.calls`, ligada a `wbot.*.*.call.*` em `wbot.commands`, com **consumidor próprio**
(goroutine e canal separados do `engine.go.commands`). `call.ready`, `call.accept`, `call.reject`, `call.end`
e `call.start` nunca esperam o laço de mensagens. O consumidor de chamadas despacha cada comando em goroutine
própria (nada de laço serial). `engine.go.commands` **deixa de** ligar-se a `call.*` para não duplicar
entrega. A ordem entre comandos de chamada e mensagens deixa de ser garantida, e não importa: domínios
independentes.
- *Alternativas*: mesma fila (o "Atender" espera atrás de qualquer envio de qualquer empresa, o débito já
  registrado); API HTTP interna estilo `groupsapi` (resposta síncrona desnecessária, mais superfície e token).

### 6. Áudio por WebSocket PCM: navegador ↔ business ↔ engine
- **Navegador**: `getUserMedia` + `AudioWorklet`; reamostra para 16 kHz mono Int16; quadros de **20 ms
  (640 bytes)**; reprodução com jitter buffer fixo (~60 ms) e descarte acima de ~200 ms de atraso para o atraso
  não acumular.
- **business**: `GET /api/v1/calls/:id/audio` (upgrade WebSocket) autenticado; valida token, empresa e que o
  usuário é o operador que assumiu a chamada. Proxy de bytes ao WebSocket interno do engine, **e é aqui que a
  gravação lê o PCM** (Decisão 7). Biblioteca de WebSocket nova: preferência `coder/websocket`.
- **engine**: **um único endpoint interno** `GET /calls/{id}/audio` (WebSocket), autenticado por
  `X-Internal-Token`, só `expose`, não sobe sem `CALLS_AUDIO_TOKEN`. Entrega o PCM a `FeedCapturedPCM` e envia
  `OnPeerAudio`. É a **única** peça direta engine↔business; o resto é RabbitMQ.
- Mensagens de controle no mesmo WebSocket (JSON, texto) levam a telemetria do engine ao business a 1 Hz.
- *Alternativas*: WebRTC direto no engine (expõe UDP e IP público, muda a topologia Swarm/Traefik); áudio por
  RabbitMQ (centenas de mensagens pequenas por segundo e latência importam).
- **Contrapartida**: TCP tem head-of-line blocking; sob perda o áudio atrasa em vez de picotar. Fila limitada
  por chamada; quadro excedente é descartado, nunca bloqueia o engine.

### 7. Gravação no business: mixagem, MP3 e S3
- **Entrada**: PCM do operador (vindo do navegador) e do contato (vindo do engine) que **já passam** pelo proxy
  de áudio do business. Cada lado alimenta um buffer; um mixador alinha por tempo de relógio do business e soma
  (com limitação de pico) em **mono 16 kHz**; lacunas viram silêncio para manter a duração.
- **Encoder**: `shine-mp3` (porte Go puro do Shine, **LGPL-2.0**) **vendorizado** em
  `business/internal/recording/shine`, com a alteração mínima de aceitar o **bitrate como parâmetro** (o original
  fixa 128 kbps = 52 MB/h; a 32 kbps são **14,4 MB/h**, medido). Mantém o `LICENSE` e um `NOTICE.md` com as
  alterações. O encoder roda **no business**, não no engine.
- **Armadilha medida**: alimentar o encoder em blocos de 320 amostras (20 ms) **corrompe** o MP3 (9 s em vez de
  5, tom errado, confirmado com um decodificador independente). Funciona com **múltiplos exatos do quadro**
  (576 amostras a 16 kHz) ou com o áudio inteiro. O gravador acumula até 576 amostras antes de codificar e
  completa o último quadro com silêncio no fechamento.
- **Armazenamento**: `ObjectStore` com chave `{tenantId}/calls/{callId}.mp3`; o banco guarda **só a chave**
  (padrão de Atividades); a URL assinada (`PresignedGetURL`, TTL curto) é gerada a cada leitura. Upload ao fim da
  chamada a partir de arquivo temporário em disco (a chamada pode durar horas: nada de acumular tudo em RAM).
  Sem S3 configurado, o modo de gravação não é oferecido.
- **Configuração**: setting por empresa `callRecordingMode` ∈ {`off` (padrão), `optional`, `auto`}. Mudar de
  `off` para outro valor exige o **aceite do texto de responsabilidade**, gravado em `callRecordingAckBy/At`.
- **Tabelas**: colunas de gravação em `CallLogs` (`recordingKey`, `recordingStatus`, `recordingDurationSec`) e
  `CallRecordingAccess` (`callId`, `userId`, `action` ∈ {`play`,`delete`}, `at`) para auditoria.
- **Decisões de produto registradas** (do dono, não recomendações): sem aviso sonoro ao contato; mesma
  permissão `calls:read` do histórico para ouvir; sem expiração automática. O sistema mantém **exclusão manual**
  (`calls:delete`) e **registro de quem ouviu**, e exibe indicador permanente de "gravando" ao operador.
- *Alternativas descartadas*: ffmpeg (+128 MiB na imagem do business e shell-out); encoder próprio a partir da
  norma ISO (dias de trabalho, licença da norma, risco alto de áudio distorcido, e sendo derivado do Shine não
  evitaria a LGPL); gravar WAV (8× maior).

### 8. Telemetria de qualidade
- **Coleta (engine, 1 Hz)**: RTT (`c2r_rtt` da oferta + intervalo `ping`→`pong`, **a validar**: o `pong` não
  carrega o `txid` do `ping`); perda e jitter recebidos (lacunas de sequência e RFC 3550 sobre timestamp RTP);
  bytes/s TX e RX; RMS do PCM nos dois sentidos; estado e reconexões do relay. Publicada como mensagem de
  controle do WebSocket de áudio e como evento `call.quality`.
- **Interpretação (business)**: indicador de 3 níveis e **índice estimado 1–5** pelo modelo E simplificado
  (R = 93,2 − Id − Ie), rotulado **"estimado"**; limites de alerta configuráveis; resumo agregado em
  `CallLogs` (médias e piores de RTT/perda/jitter e o índice) ao encerrar. **Sem série temporal.**
- **Limite declarado**: sem RTCP, não se mede a perda operador→contato; a UI diz isso. "Contato parou de enviar
  áudio" (>5 s sem pacotes) cobre o pior caso.
- Cliente: o navegador acrescenta o atraso do seu jitter buffer à exibição.

### 9. Conexões com proxy: chamadas bloqueadas (fail-closed)
A mídia sai por **UDP direto do host** (pion sem `SettingEngine`; candidato ICE para o IP do relay). O ADR 0021
exige que uma conexão com proxy **nunca** use o IP do servidor, e nenhum proxy `socks5://`/`http://` roteia UDP
genérico de forma confiável. Com proxy configurado o engine **ignora** ofertas e rejeita originar, a UI
desabilita o botão com explicação. Rotear a mídia por proxy (SOCKS5 UDP ASSOCIATE) fica como trabalho futuro e
exige validação com um proxy real.

### 10. Elegibilidade do toque, alcance por conexão e pausar
- **Elegíveis** = usuários **online** (SSE conectado), com `calls:receive` (ou alcance de empresa) e **alcance
  sobre a conexão**. Alcance sobre a conexão: alcance de empresa/plataforma; ou a conexão está ligada a uma
  fila do usuário (`whatsapp_queues` × `user_queues`). **`User.WhatsappID` não entra**: é só a conexão
  preferida ao criar ticket e não dá visibilidade de ticket (o teste de paridade com `GetScopedDB` pegou essa
  divergência do desenho original; tocar o telefone de quem não consegue abrir o ticket da chamada seria pior).
- Nova consulta reutilizável "usuários que enxergam a conexão X" no business, **espelhando** a regra de
  visibilidade de tickets (hoje só existe como SQL dentro de `GetScopedDB`). Teste de paridade garante que as
  duas regras não divirjam.
- **Presença**: o `SSEHub` ganha consulta de salas ocupadas (`HasSubscribers(room)`); o toque vai para a sala
  `user:<tenantId>:<userId>`, inscrita pelo servidor a partir do token (nunca do query).
- **Pausar**: preferência **local do navegador** (`localStorage`), aplicada no cliente e **avisada ao servidor**
  por um flag de presença, para o servidor não contar um operador pausado como elegível. Sem nenhum elegível
  não pausado, a oferta é ignorada (Decisão 4).
- **Atribuição atômica**: `UPDATE ... WHERE status='ringing' AND handledByUserId IS NULL` e checagem de
  `RowsAffected` (precedente `claimSend`); o primeiro a atender vence.

### 11. Corrigir o SSE antes de emitir eventos de chamada (pré-requisito)
Validar `?rooms=`: aceitar só `chat:<ticketId>` de ticket visível ao usuário, `tickets:<status>`,
`helpdesk-kanban` e `notification`, e **nunca** `tenant:*` alheio nem `user:*`. Sem isso, o toque e a telemetria
vazariam entre empresas. Independente das chamadas, corrige um vazamento que já existe hoje; entra como
primeira tarefa, com teste de regressão que falha no código atual.

### 12. Sinalização assíncrona, timeouts e carga
`Query()` do adaptador espera até 15 s por ack e o handler de eventos do `whatsmeow` é serial: `offer`/`accept`
são enviados em goroutine, com timeout de toque de **45 s**. O engine expõe métricas de carga (chamadas
ativas, CPU estimada, tamanho da fila de áudio) no `/health` para o operador do sistema enxergar a saturação,
já que não há teto.

### 13. Frontend
| Peça | Onde | Detalhe |
|---|---|---|
| Provedor global de chamadas | `layout/MainLayout` | assina `call.*` pela sala do usuário; toque em **qualquer tela** |
| Modal de chamada recebida | novo componente | nome, conexão, Atender/Recusar; som `sound.mp3` em loop |
| Tela de chamada ativa | novo componente | cronômetro, silenciar, encerrar, gravar, indicador "gravando", painel de qualidade |
| Painel de qualidade | dentro da tela ativa | indicador de 3 níveis + detalhes a 1 Hz + alertas |
| Botão ligar | `TicketHeader`/`TicketActionButtons` | só ticket individual, com `calls:place`; desabilitado com motivo |
| Mensagem de chamada | `MessagesList` | `mediaType: "call"`; direção, duração, resultado, player |
| Menu **Chamadas** | `SidebarNav`, rota `/calls` | `calls:read`; tabela com qualidade e player; filtro |
| Pausar chamadas | cabeçalho (`MainTopBar`) | chave local + aviso de presença |
| Configuração | Configurações | seção "Chamadas": modo de gravação + aceite; `calls:manage` |
| Cargos | tela de Cargos | permissões aparecem sozinhas (a lista vem de `GET /cargos/catalog/permissions`); descrições em pt |
| i18n | `translate/languages` | `pt`, `en`, `es` |
| Permissão de microfone | navegador | HTTPS obrigatório (já via Traefik) |

### 14. Documentação
`docs/user/calls/USING_CALLS.md` (atendente), `docs/user/calls/ENABLING_CALLS.md` (administrador: dar permissão
a um cargo em Acessos → Cargos, gravação e S3), `docs/user/plugins/MARKETPLACE_AND_MENU.md` (como o Marketplace
e o menu lateral funcionam: onde fica, `free` × `pro`, abas, o que o Ativar faz, a regra `activePlugins` do
menu e por que um plugin `free` novo não aparece sem entrada no Hub), `docs/agents/calls.md` e um ADR.

## Risks / Trade-offs

- **Banimento** (ADR 0016): chamada é caminho de protocolo mais fingerprintável que texto; protocolo não
  oficial. → Risco de cada empresa; documentado e exibido como aviso permanente, sem bloquear.
- **Gravação sem aviso ao contato e sem retenção**: é decisão do produto, mas é o ponto juridicamente mais
  sensível (LGPD, consentimento) e acumula dado sensível sem prazo. → Aceite de responsabilidade gravado por
  quem liga a gravação, indicador permanente ao operador, exclusão manual e auditoria de quem ouviu; texto de
  responsabilidade revisável pelo jurídico do dono. **Não afirmamos conformidade legal.**
- **Todo usuário de alcance de empresa passa a poder receber toque** na atualização (`RequirePermission` o
  libera sem consultar o cargo). → Alcance por conexão + "Pausar chamadas" reduzem o ruído; consta na nota de
  atualização.
- **Presença por SSE define quem toca**: sem ninguém online, a oferta é ignorada e só o celular toca. É o
  desejado, mas quem fechou a aba não recebe.
- **Sem aparelho de teste não verificamos ponta a ponta**: os testes unitários cobrem sinalização, codec e MP3
  (com decodificador independente), não uma ligação real. → Roteiro de teste manual com dois números e
  critérios de aceite; a validação real é do usuário (padrão do projeto).
- **Codec reimplementado e protocolo não documentado**: o WhatsApp pode mudar `<call>` ou o MLow sem aviso. →
  Testes com vetores reais isolam regressões do codec; falha de chamada nunca derruba a sessão (`recover` por
  chamada); a feature se desliga retirando permissões.
- **Qualidade por TCP** → jitter buffer limitado e descarte de quadro atrasado; WebRTC direto fica como
  evolução.
- **Saída UDP**: o engine precisa alcançar os relays por UDP. → Verificação e mensagem clara; documentação.
- **Sem teto e CPU**: muitas chamadas simultâneas saturam o engine (≈16% de um núcleo cada). → Métricas de
  carga no `/health`; o operador do sistema escala o engine.
- **Gravação depende do WebSocket do operador**: se ele cair, a chamada encerra (spec), então não há gravação
  parcial silenciosa.
- **Encoder LGPL no binário do business**: exige manter o aviso e disponibilizar as alterações; é compatível com
  a AGPL do projeto, mas **confirmar com o jurídico** (mesma cautela aplicada à Meta).
- **Dependências novas**: `pion/webrtc/v4` (engine), `coder/websocket` (business), `shine-mp3` vendorizado.
  Versões fixadas; `govulncheck` no CI.
- **Tamanho**: ~15 mil linhas portadas (11 mil do codec). → Pacote isolado com `NOTICE`; sem alterar o código
  portado além do import path e de correções documentadas.
- **Documentação divergente**: o `CLAUDE.md` afirma que o `business` é distroless; a imagem real é Alpine. Fora
  do escopo desta change, mas registrado para correção separada.

## Migration Plan

1. **Pré-requisito independente**: corrigir a validação de salas do SSE (Decisão 11), com teste de regressão.
2. Migração de banco **aditiva**: tabelas `CallLogs` e `CallRecordingAccess`, índice único `(tenantId, callId)`
   e `(tenantId, startedAt)`; permissões `calls:*` no catálogo; setting `callRecordingMode` (ausente = `off`).
   Nenhuma coluna existente muda.
3. Engine e business implantados **sem efeito visível**: sem operador elegível, o engine ignora ofertas e o
   celular continua tocando; a gravação nasce `off`.
4. **Habilitação gradual por permissão**: conceder `calls:receive` e `calls:place` a um cargo de teste; usuários de
   alcance de empresa já passam. Testar com dois números reais, uma conexão sem proxy.
5. Habilitar gravação só depois, por empresa, com o aceite.
6. **Rollback**: retirar as permissões `calls:*` dos cargos desliga o recurso para cargos comuns; reverter a imagem
   do engine volta a descartar ofertas. As tabelas aditivas e as gravações no S3 podem ficar. Sem migração
   destrutiva.

## Open Questions

- Limites iniciais de alerta de qualidade (proposta: RTT > 400 ms, perda > 5%, jitter > 60 ms); ajustáveis sem
  mudar o desenho.
- Bitrate do MP3 (proposta: 32 kbps mono, ≈14 MB por hora).
- Validar a medição de RTT pelo `ping`→`pong` (o `pong` não carrega o `txid` do `ping`); se não for confiável,
  a telemetria usa só o `c2r_rtt` do relay e o atraso do WebSocket.

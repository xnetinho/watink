# ADR 0031 — Chamadas de voz do WhatsApp no navegador

**Status:** Accepted (implementado; **validação com dois números reais pendente**, ver §Pendências)
**Data:** 2026-10-06

## Contexto

Operadores precisam receber e fazer chamadas de voz 1:1 do WhatsApp sem tirar o celular da mão
e sem um segundo aplicativo. O engine usa `whatsmeow`, que não implementa a sinalização nem a
mídia de chamadas. Há uma pilha pública que faz isso (**WaCalls**, MIT, `internal/voip`), mas
como serviço à parte ela duplicaria a sessão do WhatsApp (o aparelho vinculado é um só).

Restrições herdadas: engine é **adaptador burro** (regras ficam no business), multitenancy sem
RLS em worker (todo `WHERE "tenantId"` manual), ADR 0021 (conexão com proxy **nunca** usa o IP
do servidor) e ADR 0016 (risco estrutural de ban, sem remover o sinal de rede).

## Decisão

1. **Recurso nativo do core, sem plugin.** Não há Marketplace, `PluginInstallations` nem botão
   Ativar. Controle por **permissões de cargo** `calls:receive|place|read|delete|manage`, que
   **não** são anexadas a nenhum cargo existente. Alcance de empresa (`tenant`/`plataforma`) passa
   pelo `RequirePermission` sem consultar o cargo (comportamento herdado, registrado como mudança).
2. **Porte do WaCalls para `engine-go/internal/voip/`** (commit `edeb31f0`, MIT; `media/mlow` MIT de
   Rajeh Taher), preservando `LICENSE` e com `NOTICE.md`. Mudanças no código portado: só o import
   interno, mais um campo (`DeferPreaccept`) e hooks de medição, todas desligadas por padrão e em
   arquivo à parte (`callmanager_watink.go`). Adiciona `pion/webrtc/v4`; o build segue estático
   (`CGO_ENABLED=0`), compatível com a imagem distroless.
3. **Divisão engine × business.** O engine só faz sinalização, mídia e **medição**. O business decide:
   elegibilidade, atribuição, registro, interpretação da qualidade, gravação, S3, SSE.
4. **O engine nunca envia `reject` por conta própria.** `reject` é tratado pelo WhatsApp como
   recusa da **conta inteira** e derruba o toque nos outros aparelhos. Oferta ocupada, com proxy, de
   vídeo/grupo ou sem operador elegível é **ignorada** (o celular segue tocando) e vira `call.missed`.
   O `preaccept` só sai depois que o business confirma operador elegível (`call.ready`, ~3 s).
5. **Fila e consumidor dedicados** `engine.go.calls` (`wbot.*.*.call.*`), com canal próprio e uma
   goroutine por comando, e `api.events.calls.go` no business. Mensagens de uma empresa com muito
   tráfego não atrasam "Atender"/"Encerrar" de outra. `engine.go.commands` deixa de ligar `call.*`.
6. **Áudio por WebSocket PCM** navegador ↔ business ↔ engine: 16 kHz mono Int16, quadros de 20 ms
   (640 B). Um único endpoint interno no engine (`GET /calls/{id}/audio`, só `expose`; a autenticação por
   `X-Internal-Token` original foi removida na revisão abaixo). Fila limitada de 50 quadros por
   sentido com descarte do **mais antigo**: um destino lento nunca trava quem produz. Jitter buffer de
   ~60 ms no navegador, descarte acima de ~200 ms.
7. **Gravação no business**, não no engine: o PCM já atravessa o business e ele já fala com o S3.
   Mixador por relógio de 20 ms; `shine-mp3` (LGPL v2) **vendorizado** com uma única alteração (bitrate
   como parâmetro), mono 16 kHz a 32 kbps (~14,4 MB/h). O encoder consome **blocos de 576 amostras**:
   alimentá-lo com 320 corrompe o MP3 (medido, e coberto por teste que prova a armadilha). Chave
   `{tenantId}/calls/{callId}.mp3`; o banco guarda só a chave; a URL assinada (5 min) é gerada a cada
   leitura. Modos `off` (padrão), `optional`, `auto`; sair de `off` exige o **aceite** do termo, gravado
   com usuário e horário. Escuta e exclusão são auditadas (`CallRecordingAccess`); sem expiração
   automática; sem aviso sonoro ao contato (decisão do dono do produto).
8. **Conexões com proxy não fazem nem recebem chamadas** (fail-closed). A mídia sai por UDP direto do
   host e nenhum proxy `socks5://`/`http://` roteia UDP de forma confiável; usar o IP do servidor
   anularia o ADR 0021.
9. **Correção do SSE como pré-requisito.** `GET /events` aceitava qualquer sala em `?rooms=`: um usuário
   da empresa A que pedisse `tenant:<id da B>` recebia os eventos da B (reproduzido). Agora só passam
   `chat:<ticket visível ao usuário>`, `tickets:<open|pending|closed>`, `helpdesk-kanban` e
   `notification`; `tenant:*` e `user:*` são descartados e a sala pessoal `user:<tenantId>:<userId>` é
   inscrita pelo servidor a partir do token. Sem isso, o toque e a telemetria vazariam entre empresas.

## Divergências do plano original

Registradas porque o teste mostrou que o plano estava errado ou incompleto:

- **`User.WhatsappID` não dá visibilidade.** O plano dizia "a conexão é a do usuário **ou** está ligada a
  uma fila dele". O teste de paridade com `GetScopedDB("Tickets")` falhou: esse campo é só a conexão
  preferida ao criar ticket e não faz o ticket aparecer. Tocar o telefone de quem não consegue abrir o
  ticket da chamada seria pior que não tocar, então ficou de fora.
- **RTT só pelo `c2r_rtt`.** O par `ping`→`pong` do relay **não** foi validado (o `pong` não carrega o
  `txid` do `ping` de forma confirmada e o caminho de recepção descarta STUN). Fica para a chamada real.
- **Prazo de mídia (`media_timeout`).** Não estava no plano. Com a saída UDP bloqueada o relay nunca
  conectava e nada encerrava a **chamada** (só o relay desistia, aos 20 s): ficava em "Conectando…"
  ocupando a conexão. Agora há um prazo de 25 s que a encerra com o motivo e a dica da saída UDP.
- **`call.reset`.** O engine guarda chamadas só em memória; ao (re)iniciar uma sessão ele publica
  `call.reset` para o business marcar como interrompidas as chamadas que ficaram órfãs.
- **`Delete` no `ObjectStore`.** A exclusão de gravações exigiu estender a interface.

## Consequências

- Todo usuário de alcance de empresa passa a poder receber toque na atualização (nota de atualização).
- A instalação precisa de **saída UDP** a partir do engine e, para gravar, de **S3**.
- Uma chamada por conexão e uma por operador; sem teto por empresa (o limite natural é a CPU: codificar
  custa ≈16% de um núcleo por chamada, medido).
- O engine passa a depender de `pion/webrtc` e a imagem do business ganha o `shine-mp3` (LGPL) vendorizado.

## Pendências (honestas)

- **Nunca foi feita uma chamada real.** Tudo está coberto por testes com fakes, RabbitMQ e Postgres
  reais, e WebSockets reais, mas a sinalização WhatsApp, o relay UDP, o codec MLow ponta a ponta e a
  qualidade do áudio **dependem do teste manual com dois números** (tarefa 10.4 do plano).
- O risco de ban é o do ADR 0016 (fingerprint estrutural do `whatsmeow`); chamadas somam sinal novo
  que não foi medido.
- Rotear a mídia por proxy (SOCKS5 UDP ASSOCIATE) é trabalho futuro e exige validação com um proxy real.

## Revisão (2026-10-06, após o primeiro deploy)

- **O canal de áudio do engine não tem mais token.** O `X-Internal-Token`/`CALLS_AUDIO_TOKEN` foi
  removido por decisão do dono do produto: a comunicação business↔engine é interna, como o RabbitMQ
  das mensagens. Consequência aceita: **quem alcança a porta 8085 ouve e injeta áudio** de chamadas em
  andamento (precisa conhecer o `callId`). A defesa é só de rede: `expose`, nunca `ports:`, e nenhum
  outro serviço na rede do engine. O `GROUPS_API_TOKEN` (porta 8084) continua exigido.
- **Um único endereço do engine no business:** `ENGINE_HOST` (só o nome, sem esquema nem porta). Daí
  saem `/health` (8083), grupos (8084) e o áudio (`ws://…:8085`); as variáveis antigas
  (`ENGINE_HEALTH_URL`, `GROUPS_API_URL`, `CALLS_AUDIO_URL`) sobrepõem cada uma, para não quebrar
  instalações existentes. Implementado em `business/pkg/engineaddr`.
- **Sem `ENGINE_HOST` a falha deixou de ser silenciosa:** o business registra o erro da discagem ao
  engine e o painel mostra "canal de áudio indisponível" (antes a chamada era encerrada sem aviso e
  parecia "perdida").
- **O painel diz quem encerrou** (contato desligou, recusou, não atendeu, ocupado, atendida em outro
  aparelho, ou "você encerrou"); a queda do WebSocket que acompanha o fim da chamada não é mais
  mostrada como falha.

## Referências

[`docs/agents/calls.md`](../agents/calls.md) · [`docs/user/calls/`](../user/calls/) · ADR 0016, 0019, 0021, 0022

# Tasks

> Fonte técnica: `design.md` desta mudança e ADR 0032. **Regra:** a fase N+1 só começa depois de a fase N
> passar no roteiro manual com dois números reais. Convenções do repositório: `auth.GetScoped`,
> `Session(NewDB:true)`, testes contra Postgres real, swagger regenerado em rota nova, arquivos < ~250
> linhas, `lucide-react` e `src/components/ui`. Todo teste novo é provado por mutação.
> Origem do código portado, a registrar em `engine-go/internal/voip/NOTICE.md`: `purpshell/meowcaller`
> (MIT, Rajeh Taher), `fabriciosprj/WaCalls-Video` e `arthost/WaCalls` (MIT, jotadev66).

## BUG ABERTO — vídeo do operador congela de forma intermitente (20/out/2026)

O vídeo do operador chega fluido ao celular na maioria das chamadas, mas em algumas congela num quadro e
não volta. Ligar/desligar a câmera mostra sempre o mesmo quadro congelado. Tratado como bug, não como
funcionalidade pendente: a fase 2 continua marcada como entregue.

**Medido (chamada que congelou):** relays conectados, SRTCP do contato autenticado, SSRC de vídeo igual na
alocação do relay e no envio, `au_sent` 65–68 por janela de 5 s, `idr_sent` 2–3 (IDR a cada ~2 s), `pli=0`,
`au_gated=0`, nenhum erro. O engine entrega tudo ao relay.

**Já refutado (não repetir):** navegador parar de gerar IDR; descasamento de SSRC de vídeo; SPS/PPS fora de
STAP-A; reenvio do primeiro IDR; falta de NACK (nenhum PT 205 chegou do contato, não há prova de que seja a
causa).

**Corrigido no caminho (commits de 20/out):** `VideoFrameInfoDelta` 0x20 → 0x00; empacotamento `PacketizeAu`
(STAP-A para SPS+PPS, um pacote por slice); REMB agora chega ao encoder.

**Candidatos sem prova:** `short_offset` e `initial_bandwidth` sempre 0 (o Android real manda `short_offset`
29/36); ausência de FEC de vídeo (o contato manda PT 225/231, não identificados); perfil H.264 que o navegador
produz variar entre chamadas.

**Próximo passo:** capturar o que o decoder do celular registra durante o congelamento (`adb logcat` do
WhatsApp/`MediaCodec` por depuração sem fio) e comparar byte a byte um pacote de vídeo nosso com um do contato.

- [ ] B.1 Reproduzir e capturar o `logcat` do celular na chamada congelada
- [ ] B.2 Comparar cabeçalho RTP/extensão 0xdebe e NAL types: pacote nosso × pacote do contato
- [ ] B.3 Corrigir a causa provada e validar em 10 chamadas seguidas sem congelar
- [ ] B.4 Teste de regressão que cubra a causa

## 0. Base de sinalização e mídia (sem UI)

- [x] 0.1 Trocar o `<video>` do `offer`/`accept`/`preaccept` de `vp8` para H.264 (`enc="h.264" dec="H264"
      screen_width/height device_orientation`), sem o atributo `orientation`; capability de vídeo já adotada; teste
      de ordem dos filhos do `offer` e dos atributos, provado por mutação
- [x] 0.2 `OfferHasVideo` e propagar `video=true` até o `preaccept`/`accept`; teste com oferta com e sem `<video>`
- [x] 0.3 Portar `rtp/h264.go` da meowcaller: `SplitAnnexB`, `AUHasIDR`, `PackageH264NALU` (AU como um NAL, FU-A,
      payload 800), depacketizador e `H264AccessUnitAssembler` com recuperação de IDR; trazer os vetores de teste
- [x] 0.4 Cabeçalho RTP de vídeo com extensão `0xDEBE` (`MediaFrameInfo` 0x08/0x20, `FrameNumber`,
      `TransportSequence`) e `VideoRtpStream`; testes com os bytes das capturas (`rtp_test.go:89-212`)
- [x] 0.5 SSRC de vídeo com slot 2 e subscrição no relay junto do áudio; teste cruzado do `GenerateSecureSsrc`
- [x] 0.6 SRTP do vídeo: mesmas chaves, mas **estado de sequência, ROC e anti-replay por SSRC** (`SrtpSession` agora
      guarda um contexto por SSRC). Achado: com um contexto único o vídeo (seq alto) empurrava a janela do áudio e o
      derrubava como "repetido". Testes de intercalação, replay por fluxo, ROC por fluxo e pacote forjado de vídeo
- [x] 0.7 O dedup de relay (`rxLockedSsrc`) do arthost **não existe no nosso porte**, então o risco que ele tinha não se
      aplica; o demux por payload type 97 em `onRelayData` manda o vídeo ao caminho próprio e não toca na subscrição
      de áudio (`peerSsrcs`/`actualPeerSet`); teste cobre, e o anti-replay descarta as cópias do mesmo pacote vindas de
      outros relays
- [ ] 0.8 **Adiado para a fase 4.** O ack tipado `<ack class="call" type="video">` só importa no upgrade no meio da
      chamada: o changelog da meowcaller diz que "from-start video calls are unaffected" (o vídeo é negociado no
      offer/accept). Exige interceptar o `<call>` por reflection+unsafe no `nodeHandlers` do whatsmeow (marcado NOT
      VALIDATED por ela); não vale o risco para receber e iniciar. O construtor `BuildVideoAck` já existe e está testado
- [ ] 0.9 Estados in-call `0,1,3,4,5,6,8,11`: **construtores prontos e testados** (`signaling/video.go`); a máquina de
      estados (`engine_lifecycle_test.go`) fica para a fase 4

## 1. Receber vídeo

- [x] 1.1 Engine: demux do PT 97 → assembler → `OnPeerVideo`, com a AU em Annex-B e a flag de quadro-chave; lacuna de
      sequência descarta o quadro quebrado e só volta a entregar num IDR. **O PLI (pedido de quadro-chave ao contato)
      ficou para a fase 2:** exige SRTCP cifrado, que o engine ainda não tem. O gancho `OnVideoKeyframeNeeded` existe
      e é throttled a 300 ms; hoje a recuperação depende de o contato mandar um IDR sozinho
- [x] 1.2 Protocolo do canal (**revisado**): o PCM segue **intocado** (640 B, sem cabeçalho); o vídeo é uma mensagem binária
      com **prefixo mágico `FF 56 44 01`** + flags + ts90k + Annex-B, nos três pontos (engine, business, navegador),
      com vetor de bytes idêntico nos dois lados. Fila de vídeo **separada** da de áudio (32 quadros, descarta o mais
      antigo, guarda cópia). O vídeo nunca entra no gravador de áudio; vídeo vindo do navegador (fase 2) é descartado
- [x] 1.3 Business: ponte do vídeo no `ServeAudio`, mesma autorização e canal único; `call.state` indica
      `video: true`
- [x] 1.4 Frontend: `VideoDecoder` (`avc1.42E01F`, `optimizeForLatency`), canvas, só inicia no primeiro
      keyframe, reconstrói o decoder em erro; detecção de suporte e mensagem em navegador sem WebCodecs
- [x] 1.5 Frontend: painel de videochamada (vídeo do contato, controles, indicador de qualidade já existente)
- [x] 1.6 Toque indica videochamada; atender sem suporte de vídeo atende só com voz e avisa
- [x] 1.7 **Teste real com dois números: o contato liga por vídeo, o operador atende e vê a imagem** (revalidado em homologação, out/2026)

## 2. Enviar a câmera

- [x] 2.1 Engine: SRTCP (`media/srtcp.go`, rótulos 3/4/5, tag de 10 B), `media/rtcp.go` (SR+SDES, PLI, FIR), emissor no
      `CallManager` (`callmanager_video_tx.go`: gate de IDR que descarta delta até o 1º IDR e após cada PLI, AU inteira em
      um NAL FU-A, extensão 0xDEBE, SRTP do SSRC de vídeo), SR+SDES a cada 1,5 s, PLI/FIR do contato autenticado por SRTCP,
      PLI nosso quando o vídeo do contato perde pacote. Vetores de teste da meowcaller; 18 mutações do engine pegas
- [x] 2.1b Fio: o navegador sobe o vídeo no mesmo WebSocket (fila **própria** de ~2 s no business, fora do gravador e sem
      empurrar o PCM); comando de texto `camera` validado e reserializado no business; `{"type":"keyframe"}` volta ao
      navegador; teto de leitura de 512 KB nas duas pontas (um quadro-chave passa de 32 KB, o padrão do WebSocket)
- [x] 2.2 Frontend: `CameraSender` (`getUserMedia` → `MediaStreamTrackProcessor` → `VideoEncoder` annexb/realtime, 640×480,
      15 fps, teto de 300 kbps — ver 2.4 —, quadro-chave a cada 2 s e sob pedido, descarta quando o codificador atrasa);
      botão de câmera no painel; permissão negada / câmera ausente / erro do codificador com mensagem própria; só
      Chromium, com o motivo na tela
- [ ] 2.3 `device_orientation` anunciado pelo estado 1 quando o navegador girar. **Adiado:** o comando de câmera já leva a
      orientação (hoje sempre 0, a do `VideoEncoder`); detectar a rotação do aparelho/janela e reanunciar exige o estado 1
      de novo e só faz sentido depois de ver o que o contato mostra
- [x] 2.4 **Teste real: o contato vê a câmera do operador** (capturar com o diag). Tentativa ao vivo em 17/out/2026:
      **não validou o envio**. O operador não viu o botão de câmera nem prompt de permissão. Diagnóstico: no evento real o
      `<video>` fica aninhado em `offer/destination/to/video`; a detecção anterior só examinava filhos diretos, classificava a
      chamada como voz e o painel escondia o botão. A busca da mídia agora percorre os wrappers e tem regressão com essa
      árvore real; o stack da tentativa ainda não foi reconstruído/retestado após o fix.
      **Validado ao vivo em 19/out/2026** (Android do contato): o contato viu a câmera do operador, fluida, numa chamada
      inteira. O log `video tx` do engine confirma o envio — `au_in=au_sent` a ~15 quadros/s, `au_gated=0`,
      `au_not_ready=0`, `pli=0` — e mostra (`rx_pt`) que o contato manda PSFB (PT 206) a cada janela, feedback de banda que
      o engine ainda **não** trata. A lentidão anterior (≈1 quadro a cada 10 s) era o codificador a 600 kbps passando do
      teto `init_bwe.max_bitrate=350000` que o próprio contato anuncia no `<offer>`, com o excedente descartado no caminho;
      agora o encoder vai a 300 kbps. Vale para este aparelho; a instrumentação fica, e adaptar o bitrate ao feedback do
      contato é a próxima alavanca. Ver `tasks.md` 2.2 e o commit `1b2aba4b`

## 3. Iniciar chamada de vídeo

- [ ] 3.1 `POST /calls` aceita `video: true` (mesma permissão `calls:place`); comando `call.start` leva o vídeo
- [ ] 3.2 Botão de videochamada no ticket, oculto/desabilitado sem suporte do navegador ou com proxy
- [ ] 3.3 Registro da chamada guarda se foi de vídeo; histórico e mensagem do chat mostram o ícone
- [ ] 3.4 **Teste real: operador liga em vídeo e o celular toca como videochamada** (iPhone e Android)

## 4. Upgrade e downgrade no meio

- [ ] 4.1 Engine: `StartVideo` (estado 11), `AcceptVideo` (6 antes do 4 quando a câmera local está desligada),
      `RejectVideo`, `StopVideo` (6), `EnableVideo`/`DisableVideo` (1/0), com o gate de envio até o aceite
- [ ] 4.2 Business: comandos e eventos (`call.video_request`, `call.video_state`); o pedido do contato nunca é
      aceito sozinho
- [ ] 4.3 Frontend: pedido recebido com Aceitar/Recusar; botões "ligar vídeo", "desligar vídeo"
- [ ] 4.4 **Teste real: upgrade pedido pelo operador e pelo contato, aceito e recusado; downgrade nos dois lados**
      Se o ack tipado não bastar, o próximo candidato é uma resposta `<video>` explícita de aceite.

## 5. Gravação do vídeo

- [ ] 5.1 Engine entrega os access units ao business; business grava `.h264` bruto no S3
      (`{tenantId}/calls/{callId}.h264`) com o **mesmo gatilho** do áudio (`call.state active`)
- [ ] 5.2 Modelo: `videoRecordingKey`; migração e backfill; chave nunca é URL assinada
- [ ] 5.3 Exclusão apaga áudio e vídeo; acesso e exclusão continuam auditados
- [ ] 5.4 Configuração: aviso de custo de armazenamento (~270 MB/h por chamada) e de que o `.h264` não toca no
      navegador
- [ ] 5.5 Histórico mostra "vídeo gravado" e oferece o download do `.h264`

## 6. Fechamento

- [ ] 6.1 Docs do usuário (`docs/user/calls/`), `docs/agents/calls.md`, `CLAUDE.md`, ADR 0032, roteiro manual
- [ ] 6.2 Verificação completa: engine `-race`, business, frontend; `openspec validate --strict`

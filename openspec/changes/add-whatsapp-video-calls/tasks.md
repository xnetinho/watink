# Tasks

> Fonte técnica: `design.md` desta mudança e ADR 0032. **Regra:** a fase N+1 só começa depois de a fase N
> passar no roteiro manual com dois números reais. Convenções do repositório: `auth.GetScoped`,
> `Session(NewDB:true)`, testes contra Postgres real, swagger regenerado em rota nova, arquivos < ~250
> linhas, `lucide-react` e `src/components/ui`. Todo teste novo é provado por mutação.
> Origem do código portado, a registrar em `engine-go/internal/voip/NOTICE.md`: `purpshell/meowcaller`
> (MIT, Rajeh Taher), `fabriciosprj/WaCalls-Video` e `arthost/WaCalls` (MIT, jotadev66).

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
- [x] 1.7 **Teste real com dois números: o contato liga por vídeo, o operador atende e vê a imagem**

## 2. Enviar a câmera

- [ ] 2.1 Engine: `videoSender` com gate de IDR (descarta delta até o primeiro IDR e após cada PLI),
      empacotamento, SRTP e envio; **SR+SDES periódico** do vídeo; teste de vetor do relatório (74 bytes)
- [ ] 2.2 Frontend: `getUserMedia` → `MediaStreamTrackProcessor` → `VideoEncoder` (`annexb`, realtime, 15-20 fps,
      600 kbps), keyframe a cada 2 s e sob pedido; permissão de câmera com mensagem clara; liga/desliga câmera
- [ ] 2.3 `device_orientation` anunciado pelo estado 1 quando o navegador girar
- [ ] 2.4 **Teste real: o contato vê a câmera do operador** (a parte que ninguém validou; capturar com o diag)

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

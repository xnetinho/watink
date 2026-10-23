# Design

## Arquitetura

```
Navegador (Chromium)                Business (Go)                 Engine (Go)               WhatsApp
 getUserMedia                                                                               
 VideoEncoder H.264 Annex-B ──WS binário──▶ ponte ──WS interno──▶ RTP H.264 ─SRTP─▶ relays ─▶ contato
 VideoDecoder + canvas      ◀──WS binário── ponte ◀─WS interno─── depacketiza ◀─SRTP─ relays ◀─ contato
```

O engine **não tem codec de vídeo**: só empacota/desempacota H.264 Annex-B em RTP, cifra (SRTP) e repassa.
O navegador é quem codifica e decodifica (WebCodecs). Mesmo desenho do áudio PCM (ADR 0031).

## Decisões

1. **Transporte do vídeo: o mesmo WebSocket do áudio** (`/api/v1/calls/:id/audio`), sem segundo canal: mantém
   a autorização, o limite de um canal por chamada e o watchdog que já existem. Os forks usam DataChannel
   WebRTC; aqui não, porque o servidor não termina WebRTC.
2. **Como distinguir áudio de vídeo SEM mudar o áudio** (revisado na implementação): o áudio PCM é sempre
   **640 bytes** por mensagem binária, e hoje o business trata toda mensagem binária como PCM. Em vez de pôr
   um byte de tipo no áudio (que quebraria quem só fala áudio), o vídeo ganha um **prefixo mágico** de 4 bytes
   (`0xFF 'V' 'D' 0x01`), e o receptor trata como vídeo só o que começa com ele. Um quadro PCM de 640 B nunca
   começa com `FF 56 44 01` por acaso de forma ambígua porque o tamanho é diferente: PCM tem tamanho fixo e
   o vídeo é detectado por tamanho **e** prefixo. Formato: `[FF 56 44 01][flags 1B][ts90k 4B BE][AnnexB…]`.
   `flags`: bit0 = keyframe, bits1-2 = rotação. Controle servidor→navegador **em texto JSON** (o canal já
   carrega telemetria assim): `{"type":"keyframe"}` = "gere um quadro-chave agora".
3. **Codec:** H.264 Constrained Baseline nível 3.1 (`avc1.42E01F`), `avc:{format:"annexb"}`,
   `latencyMode:"realtime"`, 640×480 a 15-20 fps, 600 kbps. Keyframe a cada 2 s e sob pedido. O nível 3.1
   limita a 1280×720; resolução maior é rejeitada pelo `VideoEncoder`.
4. **RTP (referência: meowcaller `rtp/h264.go`, `rtp/rtp.go`, e o `rtph264.go` do fabriciosprj):**
   - payload type **97**, clock 90 kHz, SSRC de vídeo = `GenerateSecureSsrc(callID, deviceJid, 2)` (slot 2,
     separado do áudio, slot 0). Confirmado que o nosso `GenerateSecureSsrc` é o mesmo HKDF.
   - A **access unit inteira é fragmentada como um único NAL** em FU-A (AUD tipo 9 descartado, NALs unidos por
     `00 00 00 01`), MTU de payload **800**. Não é NAL a NAL: o WhatsApp não aceita.
   - **Extensão RTP `0xDEBE` obrigatória** com `MediaFrameInfo` (IDR `0x08`, delta `0x20`), `FrameNumber` no
     primeiro pacote da AU e `TransportSequence` por pacote. Sem isso o WhatsApp nunca trata um quadro como
     keyframe (bug real do fabriciosprj).
   - marker no último pacote da AU; timestamp igual para todos os pacotes de uma AU.
5. **SRTP do vídeo usa as mesmas chaves do áudio** (mesmo `callKey` e HKDF por JID); muda SSRC, PT e
   cabeçalho. O contexto SRTP é **por SSRC** (o IV inclui o SSRC), então a mesma `SrtpSession` serve, mas o
   ROC/replay do vídeo é independente do áudio. O nosso HKDF com sal nulo é idêntico ao sal de 32 zeros da
   meowcaller (verificado por teste).
6. **Pedido de quadro-chave ao contato (PLI) fica fora da fase 1.** O PLI é um RTCP que precisa ir cifrado
   por SRTCP com chaves e índices próprios (`UnprotectSrtcp`/`ProtectSrtcp`, tag de 10 B), que nosso engine
   ainda não tem. Na fase 1 o engine **descarta o quadro quebrado e só volta a entregar a partir do próximo IDR**
   que o próprio contato mandar (celulares mandam IDR periodicamente), e o gancho `OnVideoKeyframeNeeded`
   já existe para ligar o PLI na fase 2, quando o SRTCP for implementado para o SR+SDES.
7. **Recepção:** demux por payload type 97, `H264AccessUnitAssembler` com recuperação de keyframe: em lacuna de
   sequência descarta a AU e exige novo IDR, devolvendo `recoveryNeeded` → o engine envia PLI e o business
   avisa o navegador que descarte até o próximo keyframe. **O `rxLockedSsrc` do dedup de relay precisa ser
   por tipo de mídia (por PT)**: hoje ele trava no primeiro SSRC que decifra (o áudio) e descartaria o vídeo.
8. **Sinalização (referência: meowcaller `signaling/video.go` e `stanza.go`):**
   - `offer` com vídeo: `<video enc="h.264" dec="H264" screen_width="1920" screen_height="1080"
     device_orientation="0"/>`, sem o atributo legado `orientation`; capability `…e0 fa 13` (já adotada).
     O `vp8` atual do `signaling_build.go` **deixa de existir**: o iPhone testado ignora.
   - `preaccept` com vídeo: `<video dec="H264" device_orientation="0" screen_width="0" screen_height="0"/>`
     e capability do offer; `accept`: `<video dec="H264" device_orientation="0"/>`.
   - **Estados in-call `<call><video state=N>`:** 0 off, 1 on, 3/11 pedido de upgrade (v2 com
     `voip_settings="video"`), 4 aceite (`dec="H264,AV1"`), 5 recusa, 6 parar (downgrade, **não** encerra a
     chamada), 8 cancelar. Receber um pedido **não aceita sozinho**: o operador decide.
   - **Ack tipado obrigatório**: todo `<video>` recebido recebe `<ack class="call" type="video">`. O ack
     genérico do whatsmeow faz o outro lado cancelar o upgrade em ~5 s. Exige interceptar o `<call>` no
     engine (a meowcaller usa reflection+unsafe no `nodeHandlers` privado, marcado NOT VALIDATED por ela).
   - **Ordem dos filhos do `offer` é carga estrutural** (o servidor devolve erro 439 se errar): `privacy →
     audio 8k → audio 16k → video → net → capability → destination → encopt → device-identity`.
9. **Feedback:** PLI/FIR do contato chega por SRTCP autenticado (`RtcpRequestsKeyframe`); o engine marca
   `keyframeRequired`, descarta deltas até o próximo IDR e avisa o navegador. Para o vídeo do **chamador**
   começar a fluir é necessário enviar **SR+SDES** periódico (74 bytes protegido). REMB, NACK e
   retransmissão ficam **fora** (o fabriciosprj os tem, mas admite não conseguir autenticar o NACK de entrada).
10. **Orientação:** só pelo atributo `device_orientation` do stanza (como a meowcaller). O CVO nos bits 0-1 do
   `MediaFrameInfo` fica fora; o navegador envia sempre na orientação do `VideoEncoder`.
11. **Gravação do vídeo (decisão do dono):** o áudio segue em MP3, com o gatilho único `call.state active`. O
    **vídeo H.264 bruto (Annex-B) vai a um arquivo `.h264` à parte** no S3, `{tenantId}/calls/{callId}.h264`,
    sem muxar e sem timestamps de reprodução; serve só para auditoria. Mesma regra de modo e aceite da voz,
    mesmo `recordingKey` ganha um irmão `videoRecordingKey`. Exclusão apaga os dois. **Custo de
    armazenamento**: a 600 kbps são ~270 MB/hora por chamada. O administrador precisa saber disso antes de
    ligar o modo automático.
12. **Permissões:** as mesmas da voz. `calls:receive` atende com vídeo, `calls:place` liga com vídeo. Sem
    permissão nova. Conexão com proxy segue sem chamada (mídia por UDP direto do host).
13. **Navegador:** só Chromium (`MediaStreamTrackProcessor`/`Generator`). Outros navegadores caem em voz, e o
    botão de vídeo fica oculto, com o motivo explicado.

## Fases (com teste real obrigatório entre elas)

| Fase | Entrega | Por quê nesta ordem |
|---|---|---|
| **0** | Sinalização H.264, ack tipado, `rxLockedSsrc` por tipo, RTP H.264 + testes de vetor | Base sem a qual nada flui |
| **1** | **Receber** vídeo (atender videochamada, operador vê o contato) | Único sentido confirmado ao vivo |
| **2** | **Enviar** a câmera do operador | Nunca validado por ninguém; isolar o risco |
| **3** | **Iniciar** chamada já como vídeo | Offer H.264; resposta do iPhone ainda incerta |
| **4** | **Upgrade/downgrade** no meio da chamada | Nunca validado ao vivo, máquina de estados delicada |
| **5** | Gravação `.h264` + UI de configuração | Depende de o vídeo fluir |

**Regra:** a fase N+1 só começa depois de a fase N passar no roteiro manual com dois números. Falha numa fase
para o escopo seguinte até a causa ser entendida (a meowcaller sugere, para o upgrade, que se o ack tipado
não bastar o próximo candidato é uma resposta `<video>` explícita de aceite).

## Riscos e perguntas em aberto

- **Envio e upgrade nunca foram provados ao vivo por nenhum projeto.** Esta mudança pode precisar de várias
  rodadas de teste com dois números por fase.
- O ack tipado depende de interceptar o `<call>` no engine; a forma da meowcaller (reflection+unsafe) quebra se
  o layout do whatsmeow mudar. Preferir um ponto de extensão do próprio engine se existir.
- O empacotamento "AU inteira como um NAL" é incomum frente ao RFC 6184; está fixado por teste na meowcaller e
  confirmado na recepção ao vivo, mas precisa de captura nossa para validar o envio.
- O risco de bloqueio do número (ADR 0016) soma o sinal de vídeo, ainda não medido.
- Resolução, bitrate e fps são premissas dos forks; o ajuste real depende de teste.

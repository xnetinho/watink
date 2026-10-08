# ADR 0032 — Chamadas de vídeo do WhatsApp no navegador

**Status:** Proposed (planejado; **nada implementado**, ver `openspec/changes/add-whatsapp-video-calls`)
**Data:** 2026-10-06

## Contexto

A voz 1:1 funciona (ADR 0031). O próximo passo pedido é vídeo: receber, enviar, iniciar já como vídeo,
alternar no meio da chamada e gravar. Pesquisamos `purpshell/meowcaller` (biblioteca Go pura, MIT) e os
forks `arthost/WaCalls`, `fabriciosprj/WaCalls-Video` e `fabioangeloliongo/WaCalls`.

## Decisão

1. **Sem codec de vídeo no servidor.** O navegador (WebCodecs, Chromium) codifica e decodifica H.264; o engine
   só empacota em RTP, cifra em SRTP e repassa, mantendo o papel de adaptador burro.
2. **Mesmo canal WebSocket do áudio**, com um byte de tipo por mensagem (PCM, vídeo, pede keyframe). Os forks
   usam DataChannel WebRTC; aqui o servidor não termina WebRTC.
3. **Formato do RTP vem da meowcaller** (payload type 97, 90 kHz, SSRC slot 2, AU inteira como um único NAL em
   FU-A de 800 bytes, extensão `0xDEBE` com tipo de quadro). Os bugs que o `fabriciosprj` descobriu (extensão
   ausente, fragmentação NAL a NAL, ack não tipado, ausência de SR+SDES) entram como requisitos.
4. **Sinalização H.264** no lugar do VP8 atual (que o iPhone ignora), com ack tipado `type="video"` e os
   estados `<video state=N>`.
5. **Gravação:** áudio em MP3 como hoje; vídeo H.264 bruto em arquivo à parte (decisão do dono), sem muxer.
6. **Permissões:** as da voz. **Só Chromium.** Conexão com proxy segue sem chamada (ADR 0021).
7. **Entrega em fases com teste real obrigatório** (0 base, 1 receber, 2 enviar, 3 iniciar, 4 upgrade,
   5 gravar), porque envio e upgrade nunca foram validados ao vivo por nenhum projeto de referência.

## O que a pesquisa deixou claro

| Parte | Estado nos projetos de referência |
|---|---|
| Receber vídeo 1:1 | Confirmado ao vivo (meowcaller) |
| Enviar vídeo 1:1 | **NOT VALIDATED** (só testes sintéticos) |
| Upgrade/downgrade no meio | **NOT VALIDATED** (validado só contra whatsapp-rust) |
| Iniciar já com vídeo (1:1) | Sem registro de teste com telefone real |
| Vídeo/ações de grupo | Experimental; **fora do escopo** |

Nenhum dos projetos exige whatsmeow novo nem o fork `hypermeow` para o vídeo 1:1.

## Consequências e riscos

- Esta mudança **pode precisar de várias rodadas de teste real por fase**; o prazo não é previsível.
- O ack tipado exige interceptar o `<call>` no engine. A meowcaller usa reflection+unsafe sobre um mapa
  privado do whatsmeow, o que quebra se o layout mudar; preferir um ponto de extensão próprio.
- Armazenamento do `.h264` cru: ~270 MB/h por chamada a 600 kbps. O modo automático pode custar caro.
- O risco de bloqueio do número (ADR 0016) soma o sinal de vídeo, ainda não medido.
- Firefox e Safari ficam só com voz.

## Referências

`openspec/changes/add-whatsapp-video-calls/` · ADR 0031 · ADR 0016 · ADR 0021 ·
`purpshell/meowcaller` (`rtp/h264.go`, `rtp/rtp.go`, `signaling/video.go`, `engine_media.go`) ·
`fabriciosprj/WaCalls-Video` (`media/videoframe.go`, `client/src/lib/video-pipe.ts`)

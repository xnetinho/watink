# Código de terceiros em `internal/voip`

Este diretório é um porte da pilha de voz do WhatsApp do projeto **WaCalls**.

| Item | Valor |
|---|---|
| Origem | https://github.com/JotaDev66/WaCalls (`internal/voip`) |
| Commit | `edeb31f0427aba896639db503153b777a405eccf` (2026-06-25, v1.0.0) |
| Licença | MIT, cópia em `LICENSE.WaCalls` |
| `media/mlow` | MIT, autor Rajeh Taher, cópia em `media/mlow/LICENSE` |

## Alterações em relação à origem

- Apenas o caminho de import interno (`wacalls/internal/voip/...` para
  `github.com/alltomatos/watinkdev/engine-go/internal/voip/...`).
- Comportamento original preservado por padrão. Os testes originais foram mantidos.
- Adições do Watink, todas desligadas por padrão: o campo `CallManager.DeferPreaccept`
  (uma condicional em `HandleCallOffer`) e o arquivo `call/callmanager_watink.go`
  (`SendPreaccept` e `AbandonCall`). O engine do Watink nunca envia `reject`, e só
  envia `preaccept` depois que o business confirma operador elegível.

## Correções trazidas de forks (MIT, mesmo autor da licença)

| Mudança | Origem | O que foi feito |
|---|---|---|
| `media/srtp.go`: verificação da tag de autenticação (HMAC, tempo constante), janela anti-replay de 64 pacotes (RFC 3711), ROC só avança após autenticar, mutex | `arthost/WaCalls` (`media/srtp.go` e `srtp_{auth,replay,race}_test.go`) | Reescrito sobre o nosso `srtp.go`. Antes pacote forjado, adulterado ou repetido era aceito. Resolve também a duplicação de áudio quando o mesmo pacote chega por vários relays. |
| `media/mlow/{fft,lpc,perc,analysis}.go`: tabela de twiddle da FFT pré-calculada e buffers reutilizados | `arthost/WaCalls` commit `7e06e90` (2026-07-16) | Cópia dos arquivos do commit. Mantém a aritmética float32 original: 160 de 160 pacotes codificados idênticos aos do porte original. Codificar 60 ms caiu de ~10,0 para ~3,9 ms. |
| `call/callmanager.go`: `reject`/`terminate` enviados fora do contexto do chamador; `terminate` vai ao aparelho que atendeu; `RejectCall` devolve o erro de transição inválida | `arthost/WaCalls` commits `3123307` e `8ef1695` e `call/callmanager_endsend_test.go` | Reescrito sobre o nosso `callmanager.go`. Antes o envio era abandonado se o contexto do comando já tivesse sido cancelado. |

| `media/h264.go` (+ `h264_test.go`): packetizer FU-A, depacketizer e `H264AccessUnitAssembler` com recuperação de IDR | `purpshell/meowcaller` `rtp/h264.go` (MIT, Rajeh Taher), que o traz do WaCalls `feat/video-calls` (jotadev66) | Cópia com o pacote renomeado. A AU inteira vai como um único NAL em FU-A de 800 bytes, porque é assim que o WhatsApp a envia (fixado por teste) |
| `media/h264_rtp.go` (+ testes): extensão RTP `0xDEBE` de vídeo e `VideoRtpStream` | `purpshell/meowcaller` `rtp/rtp.go`; o `fabriciosprj/WaCalls-Video` descobriu que sem o tipo de quadro nela o WhatsApp nunca trata um quadro como keyframe | Reescrito sobre o nosso `RtpHeader`. Os testes usam os bytes de capturas reais (Android e web) |
| `signaling/video.go` (+ `video_test.go`): `<video>` H.264, estados 0..11 e ack tipado | `purpshell/meowcaller` `signaling/video.go` | Adaptado aos tipos do whatsmeow upstream. O `<video>` anterior (`vp8`) foi removido: o iPhone o ignora |
| `media/srtp.go`: estado de sequência, ROC e anti-replay por SSRC | Achado nosso, ao somar o vídeo | `SrtpSession` mantém um contexto por SSRC com as mesmas chaves |

Os testes de bit-exactness (`fft_bitexact_test.go`) falham se o cálculo do ângulo mudar.
Em `mlow/*.go` o commit também troca `for i := 0; i < n; i++` por `for i := range n`
(equivalente). `fft_bench_test.go` ganhou as duas constantes que vinham do adaptador de codec
que não portamos.

Não trazido de propósito: o estado "reconectando" com redial de relay do `arthost` (~960 linhas
em `transport` e na máquina de estados), por não haver como validá-lo sem uma chamada real.
O tratamento de `accept` vindo de outro aparelho já existe em `internal/calls`.

## Atualizar

Reaplique o porte a partir de um commit novo da origem, rode
`go test ./internal/voip/...` e atualize o commit acima. Não edite estes arquivos
para regras do Watink: as regras ficam em `internal/calls`, que usa esta pilha.

| `media/srtcp.go`, `media/rtcp.go` (+ testes): SRTCP (rótulos 3/4/5, tag de 10 B), Sender Report + SDES, PLI/FIR | `purpshell/meowcaller` `srtp/e2e.go` e `rtp/rtcp.go` (MIT, Rajeh Taher) | Reescritos sobre o nosso `SrtpContext`/`deriveSrtpKey`, sem a camada de logging. Os vetores de teste são os dela: chaves SRTCP a partir de um segredo conhecido, o SR de 28 bytes do `kats.json`, o SR+SDES de 60 bytes (74 protegido) e o PLI de 12 bytes. |
| `media/h264.go` `PackAccessUnit`, `media/h264_rtp.go` `SetTimestampStride`, `call/callmanager_video_tx.go` (envio de vídeo) | `purpshell/meowcaller` `engine_media.go` (`videoSender.protectAccessUnitLocked`) | A AU inteira vira um só NAL em FU-A, com a extensão 0xDEBE, como ela faz. O gate de IDR (descarta delta até o primeiro IDR e após cada PLI) e o laço de SR+SDES a cada 1,5 s seguem a dela. **Nenhuma das bases provou o caminho de envio ao vivo** (a própria meowcaller o marca NOT VALIDATED). |

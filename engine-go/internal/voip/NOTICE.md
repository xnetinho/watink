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

Os testes de bit-exactness (`fft_bitexact_test.go`) falham se o cálculo do ângulo mudar.
Em `mlow/*.go` o commit também troca `for i := 0; i < n; i++` por `for i := range n`
(equivalente). `fft_bench_test.go` ganhou as duas constantes que vinham do adaptador de codec
que não portamos.

## Atualizar

Reaplique o porte a partir de um commit novo da origem, rode
`go test ./internal/voip/...` e atualize o commit acima. Não edite estes arquivos
para regras do Watink: as regras ficam em `internal/calls`, que usa esta pilha.

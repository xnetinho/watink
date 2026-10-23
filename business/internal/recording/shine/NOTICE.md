# Código de terceiros em `internal/recording/shine`

Cópia do pacote `pkg/mp3` do **shine-mp3**, usado para gravar chamadas de voz em MP3
dentro do `business` (a imagem é distroless: sem ffmpeg nem shell-out).

| Item | Valor |
|---|---|
| Origem | https://github.com/braheezy/shine-mp3 (`pkg/mp3`) |
| Commit | `517c45581c50c0cb44628987d38b975c49facd2f` (2026-08-25) |
| Licença | GNU LGPL v2 (`LICENSE`, texto integral do original) |
| Obra original | Shine, encoder MP3 de ponto fixo (Gabriel Bouvigne, Pete Everett e outros) |

## Alteração em relação à origem

**Uma só**, em `layer3.go`: `NewEncoderBitrate(sampleRate, channels, bitrate)` aceita o
bitrate em kbps como parâmetro. `NewEncoder` continua igual e delega a ela com 128.
Nada mais foi tocado; o diff contra a origem deve ser só isso (`TestVendoredDiffIsOnlyBitrate`).

## Por que LGPL aqui

A LGPL v2 permite usar a biblioteca em software de licença própria desde que o código da
biblioteca (e as nossas alterações a ele) continue disponível sob a mesma licença. Este
diretório é essa cópia, com as alterações listadas acima, e o `LICENSE` acompanha o código.

## Uso

Gravação mono 16 kHz a 32 kbps. O encoder consome blocos de **576 amostras**: alimentá-lo
com outro tamanho corrompe o MP3 (medido). Quem o usa (`internal/recording`) acumula em
múltiplos de 576 e completa o último quadro no fechamento.

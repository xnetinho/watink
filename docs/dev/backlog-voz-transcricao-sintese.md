# Backlog: transcrição, síntese de voz e chamadas automatizadas

Anotado a pedido do usuário em 29/out/2026. Nada disto foi estudado nem implementado ainda.

## Objetivo

Aproveitar as chamadas de voz e os áudios que o Watink já tem para:

1. **Transcrever áudios em texto** (voz → texto).
2. **Sintetizar voz** (texto → voz).
3. **Automatizar a realização e o atendimento de chamadas telefônicas no WhatsApp** pelo flowbuilder.

## Repositórios a estudar (ambos MIT)

- https://github.com/erickythierry/transcreveAPI (transcrição)
- https://github.com/samucamg/multivozes_br_engine (síntese de voz)

Entender o funcionamento de cada um e incorporar ao sistema, possivelmente como mais um microsserviço.

## Fundação necessária

O item 3 exige uma boa fundação e o desacoplamento do código de chamadas antes de o flowbuilder usá-las.
O ponto de partida é separar o que é sinalização/mídia da chamada (engine, `internal/calls`, `internal/voip`)
do que decide o que fazer na chamada (flowbuilder), com um contrato claro entre os dois, como já é feito com `contracts/wbot`.

## Já existe hoje (ponto de partida)

- Chamadas de voz no engine (`engine-go/internal/calls`, `engine-go/internal/voip`).
- Transcrição sob demanda de áudio (`business/internal/controllers/message_transcription.go`).
- Conversão de áudio para OGG/Opus com ffmpeg (`engine-go/internal/mediameta`).
- Referência de VoIP: `waha-voip` (em `CLAUDE.md`, repositórios de referência).

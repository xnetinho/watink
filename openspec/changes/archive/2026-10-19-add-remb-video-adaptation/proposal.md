# Proposal

## Why
Durante a validação ao vivo da fase 2 de videochamadas, os logs do engine (`rx_pt`) revelaram que o aparelho do contato envia periodicamente pacotes RTCP PSFB/REMB (PT 206) contendo a estimativa de largura de banda da rede. Atualmente o engine descarta esses pacotes e o operador envia vídeo a um teto fixo de 300 kbps. Tratar o REMB permite adaptar o bitrate do encoder dinamicamente, melhorando a fluidez em redes rápidas e evitando travamentos em oscilações.

## What Changes
- **Engine Go**: Parsing de pacotes RTCP PSFB/REMB (PT 206) e extração do bitrate estimado pelo contato.
- **Canal / Business**: Repasse do evento de estimativa de largura de banda ao WebSocket.
- **Frontend**: Ajuste dinâmico do target bitrate do `VideoEncoder` via API do navegador (`encoder.configure`) com base no REMB recebido.

## Impact
- `engine-go/internal/voip/media/rtcp.go` e `callmanager_video_tx.go`
- `business/internal/calls/audio_ws.go` (ou equivalente de vídeo/comandos)
- `frontend/src/lib/calls/cameraSender.ts`

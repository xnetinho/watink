# Tasks

- [ ] 1. Engine: adicionar parser de REMB em `internal/voip/media/rtcp.go` e testes correspondentes
- [ ] 2. Engine: tratar PT 206 (PSFB/REMB) em `callmanager_video_tx.go` / `callmanager_media.go` e expor bitRate estimado
- [ ] 3. Business: repassar evento de largura de banda estimada via WebSocket de chamadas
- [ ] 4. Frontend: receber comando de bitrate e atualizar o `VideoEncoder` dinamicamente em `CameraSender`
- [ ] 5. Testes completos: engine (`go test ./...`) e frontend (`vitest`) + validação OpenSpec (`openspec validate add-remb-video-adaptation --strict`)

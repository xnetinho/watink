# Design

## Architecture
1. **Engine**: `media.ParseRemb(data)` examina pacotes RTCP PSFB (PT 206) ou general-FCI app packets para extrair o bitRate (bps).
2. **CallManager**: Mantém `peerEstimatedBitrate uint32`. Quando atualizado, dispara notificação ou é consultado pelo laço de envio.
3. **Business / WS**: Envia comando de bitrate ao navegador quando o engine notificar mudança relevante (ex: variação > 50 kbps).
4. **Frontend (`CameraSender`)**: Recebe mensagem do WS e chama `encoder.configure({ bitrate: newBitrate, ... })`.

## Data Flow
Contato → Relay SCTP → Engine `onRtcp` (PT 206) → Parse REMB → Business WS → Frontend `cameraSender` → `VideoEncoder.configure`.

## Testing
- Testes unitários no engine para o parser de REMB (vetores simulados).
- Testes unitários no frontend para o recebimento de comando de bitrate e reconfiguração do encoder.

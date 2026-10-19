# Design

## Architecture
1. **DeviceManager (`src/lib/calls/deviceManager.ts`)**: Encapsula `navigator.mediaDevices.enumerateDevices()` e persistência de IDs preferidos em `localStorage`.
2. **VirtualBackground (`src/lib/calls/virtualBackground.ts`)**: Processador de vídeo via `HTMLCanvasElement` / Canvas 2D / WebGL para aplicar filtros antes de alimentar o `MediaStreamTrackProcessor`.
3. **UI (`DeviceSettingsModal.tsx`)**: Modal de configuração acessível na barra de controles da chamada (`ActiveCallPanel`).

## Testing
- Testes unitários para o gerenciador de dispositivos e persistência.
- Testes de renderização do modal de configuração.

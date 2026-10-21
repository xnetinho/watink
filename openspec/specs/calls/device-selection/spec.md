# Spec: calls/device-selection
Gerenciamento de periféricos de áudio e vídeo em chamadas.

## Requirements
- O sistema deve listar todos os microfones, câmeras e saídas de áudio disponíveis no navegador via `navigator.mediaDevices.enumerateDevices()`.
- O operador deve poder trocar o microfone e a câmera ativos sem reiniciar a chamada.
- O operador deve poder escolher o alto-falante/fone de ouvido (`setSinkId`) para a reprodução de áudio da chamada.

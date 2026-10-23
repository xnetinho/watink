# Proposal: Seleção de Dispositivos e Plano de Fundo Virtual nas Chamadas

## Why
Atualmente, o Watink utiliza os dispositivos padrão do sistema (microfone, câmera e alto-falante) e transmite o fundo real da câmera do operador. Para equiparar a experiência a plataformas corporativas como o Google Meet e garantir privacidade e controle profissional ao operador, precisamos permitir a escolha ativa de periféricos de áudio/vídeo e a aplicação de plano de fundo virtual (desfoque ou imagem).

## What Changes
- **Seleção de Periféricos**: Listagem e escolha em tempo real de Microfone, Fone/Alto-falante (`setSinkId`) e Câmera no painel de chamada e modal de pré-chamada.
- **Plano de Fundo Virtual**: Opções de câmera limpa, desfoque de fundo (*blur*) e seleção de imagem estática de fundo usando processamento local em Canvas/WebCodecs.
- **Persistência de Preferências**: Armazenamento local das escolhas do operador (`localStorage`).

## Capabilities
### New Capabilities
- `calls/device-selection`: Gerenciamento e listagem de periféricos de mídia de áudio e vídeo.
- `calls/virtual-background`: Processamento de canvas para desfoque e substituição de fundo na câmera do operador.

## Impact
- Frontend (`src/components/Calls/`, `src/lib/calls/`)
- Gerenciamento de estado de mídia no navegador (`MediaStream` e `AudioContext`).

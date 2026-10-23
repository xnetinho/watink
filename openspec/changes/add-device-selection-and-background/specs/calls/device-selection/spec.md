## ADDED Requirements

### Requirement: Listagem e Seleção de Periféricos
The system MUST list available microphones, cameras, and audio output devices and allow dynamic switching.

#### Scenario: Operador escolhe novo microfone
- **Given** que o painel de configuração de dispositivos está aberto
- **When** o operador seleciona um microfone diferente na lista
- **Then** a chamada passa a capturar áudio do novo microfone selecionado

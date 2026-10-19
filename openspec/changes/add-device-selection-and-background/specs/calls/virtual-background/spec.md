## ADDED Requirements

### Requirement: Efeitos de Plano de Fundo na Câmera
The system MUST provide options to blur or replace the camera background.

#### Scenario: Operador ativa desfoque de fundo
- **Given** que a câmera está ligada
- **When** o operador escolhe a opção de desfoque de fundo (*blur*)
- **Then** o canvas processa a imagem da câmera aplicando desfoque no fundo antes da transmissão

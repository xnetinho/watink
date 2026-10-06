# Spec Delta

## Purpose

Permite que atendentes recebam, façam e alternem chamadas de vídeo 1:1 do WhatsApp pelo navegador (Chromium),
com a imagem codificada e decodificada no navegador, e o engine apenas empacotando e repassando.

## ADDED Requirements

### Requirement: Receber chamada de vídeo
O sistema SHALL identificar uma oferta de chamada com vídeo, avisar o atendente que é videochamada e, ao
atender, exibir o vídeo do contato junto ao áudio.

#### Scenario: Oferta com vídeo é identificada
- **WHEN** o contato liga por vídeo
- **THEN** o toque indica videochamada e, ao atender, o atendente vê a imagem do contato e ouve o áudio

#### Scenario: Navegador sem suporte atende só com voz
- **WHEN** o atendente usa um navegador sem WebCodecs de vídeo e atende uma videochamada
- **THEN** a chamada é atendida só com áudio e o painel informa que o vídeo exige Chrome, Edge ou Brave

#### Scenario: Perda de pacotes de vídeo é recuperada
- **WHEN** a sequência de vídeo recebida tem uma lacuna
- **THEN** o sistema descarta o quadro incompleto, pede um novo quadro-chave ao contato e retoma a exibição
  no próximo quadro-chave, sem travar o áudio

### Requirement: Enviar câmera
O sistema SHALL permitir que o atendente ligue e desligue a câmera durante uma chamada de vídeo, enviando a
imagem ao contato, e SHALL pedir permissão de câmera ao navegador.

#### Scenario: Câmera negada não derruba a chamada
- **WHEN** o atendente nega a permissão de câmera
- **THEN** a chamada continua só com áudio e o painel explica como liberar a câmera

#### Scenario: Contato pede quadro-chave
- **WHEN** o contato envia um pedido de quadro-chave
- **THEN** o navegador do atendente gera um quadro-chave imediatamente e o envia

### Requirement: Iniciar chamada já como vídeo
O sistema SHALL oferecer um botão de videochamada no ticket, ao lado do de voz, que inicia a chamada já com
vídeo, exigindo a mesma permissão `calls:place`.

#### Scenario: Botão de vídeo só onde há suporte
- **WHEN** o navegador não suporta o envio de vídeo, ou a conexão tem proxy
- **THEN** o botão de videochamada fica desabilitado e informa o motivo

### Requirement: Upgrade e downgrade no meio da chamada
O sistema SHALL permitir transformar uma chamada de voz em vídeo e voltar a voz durante a conversa, com pedido
e aceite de qualquer lado, e SHALL NOT aceitar um pedido do contato sem decisão do atendente.

#### Scenario: Contato pede vídeo
- **WHEN** o contato pede para ligar o vídeo no meio de uma chamada de voz
- **THEN** o atendente vê o pedido e escolhe aceitar ou recusar; nada muda sem a escolha

#### Scenario: Atendente pede vídeo
- **WHEN** o atendente pede vídeo e o contato aceita
- **THEN** o vídeo passa a fluir nos dois sentidos sem interromper o áudio

#### Scenario: Voltar para voz não encerra a chamada
- **WHEN** qualquer lado desliga o vídeo
- **THEN** a chamada continua só com áudio

### Requirement: Reconhecimento tipado do estado de vídeo
O sistema SHALL responder a todo aviso de estado de vídeo recebido com o reconhecimento tipado esperado pelo
WhatsApp, para que o outro lado não cancele a mudança por falta de resposta.

#### Scenario: Pedido de upgrade reconhecido
- **WHEN** chega um aviso de estado de vídeo
- **THEN** o engine responde com o reconhecimento de tipo "video" imediatamente

### Requirement: Gravação do vídeo
O sistema SHALL, quando a gravação estiver ativa, guardar o áudio em MP3 como na voz e o vídeo H.264 bruto em
arquivo à parte no armazenamento de objetos, e SHALL apagar os dois na exclusão da gravação.

#### Scenario: Gatilho único com a voz
- **WHEN** a mídia da chamada de vídeo conecta
- **THEN** a gravação do áudio e a do vídeo começam juntas, com o mesmo gatilho do cronômetro

#### Scenario: Chamada de vídeo sem gravação ligada
- **WHEN** a gravação da empresa está desligada
- **THEN** nem o áudio nem o vídeo são guardados

#### Scenario: Excluir apaga os dois arquivos
- **WHEN** o usuário exclui a gravação de uma chamada de vídeo
- **THEN** o áudio e o vídeo são removidos do armazenamento e o histórico marca a gravação como excluída

### Requirement: Mesmo controle de acesso da voz
O sistema SHALL controlar a chamada de vídeo pelas permissões já existentes, sem criar nenhuma nova.

#### Scenario: Sem permissão de ligar
- **WHEN** um usuário sem `calls:place` abre o ticket
- **THEN** nem o botão de voz nem o de vídeo aparecem

#### Scenario: Conexão com proxy
- **WHEN** a conexão do ticket tem proxy configurado
- **THEN** não há chamada de vídeo nem de voz por ela, e o motivo é informado

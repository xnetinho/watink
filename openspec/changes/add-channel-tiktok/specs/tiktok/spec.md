# Spec Delta

## Purpose

Permite atender clientes que enviam mensagens diretas a uma conta Business do TikTok dentro do Watink.

## ADDED Requirements

### Requirement: Conectar a conta Business
O sistema SHALL conectar uma conta Business do TikTok por OAuth e sinalizar quando for preciso reconectar.

#### Scenario: Conexão
- **WHEN** o administrador autoriza a conta
- **THEN** a conexão fica ativa

#### Scenario: Região sem suporte
- **WHEN** a conta pertence a uma região em que a API de mensagens não está disponível
- **THEN** a conexão é recusada com o motivo

### Requirement: Receber mensagens
O sistema SHALL registrar mensagens diretas recebidas, deduplicá-las e recuperar as perdidas por listagem.

#### Scenario: Mensagem recebida
- **WHEN** um usuário envia uma DM à conta
- **THEN** um ticket é aberto ou atualizado

#### Scenario: Evento perdido
- **WHEN** um webhook não foi processado
- **THEN** o reparo por listagem recupera a mensagem sem duplicá-la

### Requirement: Limites de resposta por conversa
O sistema SHALL aplicar no servidor os limites de mensagens por conversa e por janela do TikTok e SHALL informar ao
atendente quantas mensagens restam.

#### Scenario: Limite atingido
- **WHEN** o atendente já enviou o máximo permitido sem nova resposta do usuário
- **THEN** o composer fica desabilitado com o motivo e a API recusa o envio

#### Scenario: Usuário respondeu
- **WHEN** o usuário envia nova mensagem
- **THEN** o limite é renovado conforme a regra da plataforma

### Requirement: Somente responder
O sistema SHALL NOT iniciar conversa com quem não escreveu antes.

#### Scenario: Novo ticket
- **WHEN** o atendente tenta abrir conversa nova no TikTok
- **THEN** a ação não é oferecida

# Spec Delta

## Purpose

Permite atender clientes pelo Telegram por meio de um bot que a própria empresa cria.

## ADDED Requirements

### Requirement: Conectar um bot por token
O sistema SHALL conectar um bot do Telegram a partir do token, validá-lo e registrar o recebimento de mensagens.

#### Scenario: Token válido
- **WHEN** o administrador cola um token válido
- **THEN** o sistema mostra o nome do bot e a conexão fica ativa

#### Scenario: Token inválido
- **WHEN** o token é inválido ou foi revogado
- **THEN** a conexão não é criada e o motivo é mostrado, sem exibir o token

### Requirement: Receber e enviar mensagens
O sistema SHALL receber mensagens de texto e mídia e responder em texto e mídia, dentro dos limites do Telegram.

#### Scenario: Mensagem recebida
- **WHEN** um usuário escreve ao bot
- **THEN** um ticket é aberto ou atualizado com a mensagem e o contato identificado pelo `chat.id`

#### Scenario: Resposta longa
- **WHEN** a resposta excede 4096 caracteres
- **THEN** o envio é recusado com o limite informado

### Requirement: Somente responder
O sistema SHALL NOT oferecer iniciar conversa com quem nunca escreveu ao bot.

#### Scenario: Novo ticket
- **WHEN** o atendente tenta criar uma conversa nova no Telegram
- **THEN** a ação não é oferecida

### Requirement: Token protegido
O sistema SHALL guardar o token cifrado e nunca devolvê-lo nem registrá-lo em log.

#### Scenario: Listagem
- **WHEN** a API lista a conexão
- **THEN** o token não aparece na resposta

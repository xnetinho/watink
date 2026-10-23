# Spec Delta

## Purpose

Permite atender compradores que escrevem pelo chat de anúncios da OLX dentro do Watink.

## ADDED Requirements

### Requirement: Conectar a conta do anunciante
O sistema SHALL conectar a conta do anunciante por OAuth, ativar o recebimento das mensagens e desativá-lo ao desconectar.

#### Scenario: Conexão
- **WHEN** o anunciante autoriza o acesso
- **THEN** o recebimento de mensagens fica ativo para a conta

#### Scenario: Desconexão
- **WHEN** a conexão é removida
- **THEN** o recebimento de mensagens é desativado na OLX

### Requirement: Receber mensagens do comprador
O sistema SHALL registrar cada mensagem do comprador e preencher o contato com nome, e-mail e telefone informados.

#### Scenario: Mensagem do comprador
- **WHEN** um comprador escreve no chat de um anúncio
- **THEN** um ticket é aberto ou atualizado, com o contato e o anúncio de origem

#### Scenario: Eco de mensagem do vendedor
- **WHEN** chega um evento originado pelo vendedor ou pelo sistema da OLX
- **THEN** ele não gera mensagem duplicada

### Requirement: Uma conversa por anúncio
O sistema SHALL abrir um ticket por anúncio e chat.

#### Scenario: Dois anúncios
- **WHEN** o mesmo comprador escreve sobre dois anúncios
- **THEN** são abertos dois tickets

### Requirement: Responder somente com texto
O sistema SHALL responder pelo chat com texto e SHALL NOT oferecer mídia nem iniciar conversa.

#### Scenario: Anexo
- **WHEN** o atendente tenta anexar um arquivo
- **THEN** a ação não é oferecida

### Requirement: Webhook protegido
O sistema SHALL aceitar o webhook somente com o segredo da conexão e SHALL permitir restringir a origem por IP.

#### Scenario: Segredo inválido
- **WHEN** o webhook chega sem o segredo correto
- **THEN** é recusado sem revelar se a conexão existe

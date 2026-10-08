# Spec Delta

## Purpose

Permite atender clientes que enviam mensagens diretas a uma conta profissional do Instagram dentro do Watink.

## ADDED Requirements

### Requirement: Conectar a conta profissional
O sistema SHALL conectar uma conta profissional do Instagram por OAuth, renovar o acesso antes de vencer e sinalizar quando for preciso reconectar.

#### Scenario: Conexão
- **WHEN** o administrador autoriza a conta
- **THEN** a conexão fica ativa e o recebimento de mensagens é assinado

#### Scenario: Acesso prestes a vencer
- **WHEN** o acesso está perto do vencimento
- **THEN** o sistema o renova sem intervenção; se falhar, a conexão passa a "reconectar"

### Requirement: Webhook autêntico
O sistema SHALL aceitar eventos somente com assinatura válida e SHALL responder ao desafio de verificação.

#### Scenario: Assinatura inválida
- **WHEN** um evento chega com assinatura inválida
- **THEN** é recusado sem processar

### Requirement: Janela de resposta de 24 horas
O sistema SHALL permitir responder em até 24 horas da última mensagem do usuário e, por ação de um atendente humano, até
7 dias; fora disso SHALL recusar com o motivo.

#### Scenario: Dentro da janela
- **WHEN** o usuário escreveu há menos de 24 horas
- **THEN** o atendente pode responder

#### Scenario: Entre 24 horas e 7 dias
- **WHEN** a janela de 24 horas expirou e passaram menos de 7 dias
- **THEN** só um atendente humano pode responder, e uma automação não

#### Scenario: Acima de 7 dias
- **WHEN** passaram mais de 7 dias
- **THEN** o composer fica desabilitado com o motivo e a API recusa

### Requirement: Somente responder
O sistema SHALL NOT iniciar conversa com quem não escreveu antes.

#### Scenario: Novo ticket
- **WHEN** o atendente tenta abrir conversa nova no Instagram
- **THEN** a ação não é oferecida

### Requirement: Limites do canal
O sistema SHALL limitar o texto a 1000 bytes e SHALL aceitar apenas os tipos de mídia suportados.

#### Scenario: Texto longo
- **WHEN** a resposta excede 1000 bytes
- **THEN** o envio é recusado com o limite informado

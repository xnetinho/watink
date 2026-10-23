# Spec Delta

## Purpose

Permite atender compradores do Mercado Livre (pós-venda e perguntas de anúncio) dentro do Watink.

## ADDED Requirements

### Requirement: Conectar a conta do vendedor
O sistema SHALL conectar a conta do vendedor por OAuth, renovar o acesso automaticamente e sinalizar quando for preciso reconectar.

#### Scenario: Conexão
- **WHEN** o vendedor autoriza o acesso
- **THEN** a conexão fica ativa com a conta identificada

#### Scenario: Acesso revogado
- **WHEN** a renovação do acesso falha
- **THEN** a conexão passa a "reconectar" e o atendente é avisado

### Requirement: Uma conversa por pedido
O sistema SHALL abrir um ticket por pedido (pack) no pós-venda e mostrar o contexto do pedido.

#### Scenario: Compra repetida
- **WHEN** o mesmo comprador escreve sobre outro pedido
- **THEN** um novo ticket é aberto para esse pedido

### Requirement: Só responder, dentro dos limites do Mercado Livre
O sistema SHALL NOT iniciar conversa com o comprador, SHALL limitar a resposta a 350 caracteres no charset ISO-8859-1 e
SHALL respeitar o bloqueio por prazo e por estado do pedido.

#### Scenario: Texto longo
- **WHEN** a resposta excede 350 caracteres ou contém caractere fora do charset
- **THEN** o envio é recusado com o motivo

#### Scenario: Conversa bloqueada
- **WHEN** o pedido está cancelado, em mediação ou o prazo de resposta expirou
- **THEN** o composer aparece desabilitado com o motivo e a API recusa o envio

### Requirement: Entrega confiável
O sistema SHALL responder ao webhook rapidamente, processar de forma assíncrona, deduplicar por identificador da
mensagem e recuperar eventos perdidos.

#### Scenario: Evento perdido
- **WHEN** um webhook não foi processado
- **THEN** o job de reparo recupera a mensagem sem duplicá-la

### Requirement: Agente de mensageria do Mercado Livre
O sistema SHALL endereçar o envio ao agente de mensageria do país configurado e SHALL identificar o comprador pelo pedido.

#### Scenario: Envio no Brasil
- **WHEN** uma resposta é enviada para o MLB
- **THEN** o destinatário é o agente configurado e o comprador é identificado pelo pedido

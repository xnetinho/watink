# Spec Delta

## Purpose

Define o comportamento único de conexões, mensagens e envio para qualquer canal, de modo que um canal novo não exija alterar o núcleo.

## ADDED Requirements

### Requirement: Toda conexão é de um tipo de canal, com identificador único
O sistema SHALL representar toda conexão de mensageria, inclusive WhatsApp, como uma conexão de canal com um identificador único por empresa, sem identificadores paralelos por tipo.

#### Scenario: Conexão WhatsApp e Telegram na mesma empresa
- **WHEN** a empresa tem uma conexão WhatsApp e uma Telegram
- **THEN** cada uma tem um identificador distinto e nenhuma referência do sistema depende de qual tipo ela é

### Requirement: Mensagens de conexões diferentes nunca colidem
O sistema SHALL identificar uma mensagem pela empresa, pela conexão e pelo identificador externo, e nunca descartar uma mensagem por coincidência de identificador externo com outra conexão ou empresa.

#### Scenario: Mesmo identificador externo em dois chats
- **WHEN** duas conversas diferentes recebem mensagens com o mesmo identificador externo
- **THEN** as duas mensagens são gravadas, cada uma na sua conversa

#### Scenario: Reentrega da mesma mensagem
- **WHEN** o canal entrega duas vezes a mesma mensagem da mesma conexão
- **THEN** apenas uma é gravada

### Requirement: Um único caminho de saída
O sistema SHALL enviar toda mensagem de saída, venha de atendente, fluxo, assistente, campanha, resposta rápida ou plugin, pelo mesmo serviço, que aplica a regra de resposta e as capacidades do canal.

#### Scenario: Resposta a ticket do Telegram
- **WHEN** o atendente responde um ticket originado no Telegram
- **THEN** a mensagem chega ao chat do Telegram e é registrada no ticket

#### Scenario: Fluxo respondendo em outro canal
- **WHEN** um fluxo automático responde uma conversa do Telegram
- **THEN** a resposta sai pelo Telegram, não pelo WhatsApp

#### Scenario: Excede a capacidade do canal
- **WHEN** a mensagem passa do limite de texto ou usa mídia que o canal não aceita
- **THEN** o envio é recusado com o motivo e o limite, sem enviar parcialmente

### Requirement: A regra de resposta do canal é respeitada
O sistema SHALL bloquear respostas fora da janela ou das regras do canal e informar o motivo ao atendente.

#### Scenario: Janela de resposta expirada
- **WHEN** o canal exige resposta dentro de uma janela e ela expirou
- **THEN** o envio é recusado com o motivo e a ação alternativa

### Requirement: Um novo canal não altera o núcleo
O sistema SHALL permitir adicionar um canal implementando apenas o contrato de canal e passando na suíte de contrato, sem alterar controllers, serviços de envio, fluxos nem frontend de tickets.

#### Scenario: Canal de teste
- **WHEN** um canal de teste implementa o contrato e é registrado
- **THEN** conectar, receber, enviar, mídia, status e regra de resposta funcionam de ponta a ponta sem nenhuma alteração fora do canal

### Requirement: O contato é identificado por canal
O sistema SHALL identificar o contato pela identidade no canal, tratando o telefone como atributo opcional.

#### Scenario: Dois contatos sem telefone
- **WHEN** dois usuários do Telegram escrevem sem compartilhar telefone
- **THEN** cada um é um contato distinto

### Requirement: Falhas de canal são visíveis
O sistema SHALL registrar o último erro de cada conexão e exibi-lo ao administrador.

#### Scenario: Token revogado
- **WHEN** o canal rejeita a credencial
- **THEN** a conexão mostra o erro e o estado desconectado

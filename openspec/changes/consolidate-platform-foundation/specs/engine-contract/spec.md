# Spec Delta

## Purpose

Define o contrato versionado e resiliente entre o engine do WhatsApp e o business, para que nenhum evento ou comando se perca em silêncio.

## ADDED Requirements

### Requirement: O contrato tem fonte única
O sistema SHALL definir comandos, eventos, formatos e endereçamento do engine em um único lugar compartilhado pelos dois lados.

#### Scenario: Evento sem destinatário
- **WHEN** o engine passa a emitir um evento que o business não trata
- **THEN** a verificação automática falha

#### Scenario: Escuta sem emissor
- **WHEN** o business passa a escutar um evento que o engine não emite
- **THEN** a verificação automática falha

### Requirement: Eventos recebidos não se perdem
O sistema SHALL preservar eventos de mensagem recebida mesmo quando o broker está indisponível, entregando-os quando ele voltar.

#### Scenario: Broker indisponível
- **WHEN** uma mensagem chega ao WhatsApp enquanto o broker está fora
- **THEN** o evento é retido e entregue depois, sem duplicar a mensagem

### Requirement: Comandos com falha recuperável são reprocessados
O sistema SHALL reprocessar comandos que falham de forma recuperável e enviar os irrecuperáveis a uma fila de falhas, sem confirmá-los como bem-sucedidos.

#### Scenario: Falha transitória no envio
- **WHEN** um envio falha por erro transitório
- **THEN** o comando é reprocessado e o resultado final é informado

#### Scenario: Comando inválido
- **WHEN** o comando é malformado
- **THEN** vai para a fila de falhas e não trava os demais

### Requirement: O consumo se restabelece sozinho
O sistema SHALL retomar o consumo de comandos após queda e retorno do broker, sem reinício manual.

#### Scenario: Queda e retorno do broker
- **WHEN** o broker cai e volta
- **THEN** o engine volta a consumir comandos e a publicar eventos

### Requirement: Uma sessão pertence a uma empresa
O sistema SHALL identificar a sessão pela empresa e pela conexão, de modo que sessões de empresas diferentes nunca se confundam.

#### Scenario: Mesmo número de sessão em duas empresas
- **WHEN** duas empresas têm sessões com o mesmo identificador numérico
- **THEN** cada comando atinge apenas a sessão da empresa indicada

### Requirement: O engine sobe sem consultar o esquema do negócio
O sistema SHALL restabelecer as sessões ativas por troca de mensagens com o business, e não por consulta direta ao banco.

#### Scenario: Reinício do engine
- **WHEN** o engine reinicia
- **THEN** informa que está pronto e o business indica as sessões a restabelecer

### Requirement: Comandos de chamada não esperam mensagens
O sistema SHALL processar comandos de chamada em fila e consumo próprios, sem espera atrás do envio de mensagens.

#### Scenario: Atender durante tráfego alto
- **WHEN** há muitas mensagens pendentes e o operador atende uma chamada
- **THEN** o comando de atender é executado sem esperar as mensagens

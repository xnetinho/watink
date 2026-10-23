# Spec Delta

## Purpose

Define os portões de qualidade, a observabilidade e a resiliência operacional que mantêm a plataforma confiável.

## ADDED Requirements

### Requirement: O CI reprova regressões de concorrência e contrato
O sistema SHALL executar, a cada alteração, os testes dos dois serviços com detector de corrida, os testes de contrato e os de arquitetura, e reprovar a alteração se algum falhar.

#### Scenario: Corrida introduzida
- **WHEN** uma alteração introduz uma corrida de dados no engine
- **THEN** o CI reprova

### Requirement: O núcleo não depende de adaptadores
O sistema SHALL reprovar alterações em que o núcleo importa um adaptador de canal concreto ou usa nomes específicos de um canal.

#### Scenario: Controller importando adaptador
- **WHEN** um controller importa o pacote de um canal
- **THEN** a verificação automática falha

### Requirement: Falhas de execução não derrubam o processo
O sistema SHALL conter pânico em consumidores e tarefas em segundo plano, registrar o erro e continuar atendendo.

#### Scenario: Pânico em tratador de evento
- **WHEN** um tratador de evento entra em pânico
- **THEN** o erro é registrado, o evento segue para a fila de falhas e o processo continua

### Requirement: Métricas e prontidão reais
O sistema SHALL expor métricas de mensagens, latência, erros por canal, filas e chamadas ativas, e indicar prontidão apenas quando banco, broker e cache respondem.

#### Scenario: Broker fora
- **WHEN** o broker está indisponível
- **THEN** a verificação de prontidão informa que o serviço não está pronto

### Requirement: Portas internas não são publicadas
O sistema SHALL reprovar configurações em que as portas internas do engine são publicadas ao exterior.

#### Scenario: Porta publicada por engano
- **WHEN** um arquivo de composição publica a porta de áudio das chamadas
- **THEN** a verificação automática falha

### Requirement: Imagens de build são fixas
O sistema SHALL construir as imagens com versões fixas, sem versão candidata nem etiqueta móvel.

#### Scenario: Imagem sem versão fixa
- **WHEN** um Dockerfile usa `latest` ou versão candidata
- **THEN** a verificação automática falha

### Requirement: A documentação reflete o código
O sistema SHALL manter tarefas e decisões consistentes com o que está implementado.

#### Scenario: Tarefa marcada sem estar feita
- **WHEN** uma tarefa está concluída mas o comportamento descrito não existe no código
- **THEN** a revisão reabre a tarefa

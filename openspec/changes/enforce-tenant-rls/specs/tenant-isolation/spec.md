# Spec Delta

## Purpose

Garante que uma empresa nunca leia nem grave dados de outra, mesmo quando uma consulta da aplicação esquece o filtro de empresa.

## ADDED Requirements

### Requirement: O banco impõe o isolamento entre empresas
O sistema SHALL restringir toda leitura e escrita de tabela com dados de empresa à empresa do contexto atual, por mecanismo do próprio banco, e não apenas por filtros da aplicação.

#### Scenario: Consulta sem filtro de empresa
- **WHEN** uma requisição da empresa A executa uma consulta que não filtra por empresa
- **THEN** o resultado contém apenas linhas da empresa A

#### Scenario: Escrita para outra empresa
- **WHEN** uma requisição da empresa A tenta inserir ou alterar uma linha com o identificador da empresa B
- **THEN** a operação é recusada e nada é gravado

### Requirement: Sem contexto de empresa, nada é acessível
O sistema SHALL tratar a ausência de empresa definida como acesso negado: nenhuma linha visível e nenhuma escrita permitida.

#### Scenario: Conexão sem empresa definida
- **WHEN** uma consulta roda numa conexão que não definiu a empresa
- **THEN** ela retorna zero linhas e qualquer escrita é recusada

#### Scenario: Reuso de conexão
- **WHEN** uma conexão termina uma requisição da empresa A e atende a próxima sem empresa definida
- **THEN** nenhum dado da empresa A permanece acessível

### Requirement: O acesso fora da empresa é explícito e restrito
O sistema SHALL permitir acesso entre empresas apenas por um caminho de sistema nomeado, limitado a operações que não pertencem a uma empresa (autenticação, provisionamento, descoberta da empresa de um webhook, varreduras de manutenção).

#### Scenario: Login
- **WHEN** um usuário se autentica e a empresa ainda não é conhecida
- **THEN** a consulta de credencial usa o caminho de sistema e o token emitido já carrega a empresa

#### Scenario: Uso indevido do caminho de sistema
- **WHEN** um pacote fora da lista autorizada tenta usar o caminho de sistema
- **THEN** a verificação automática do repositório falha

### Requirement: O papel do aplicativo não altera o esquema
O sistema SHALL executar requisições com um papel que não é dono dos objetos, não é superusuário e não ignora políticas de linha.

#### Scenario: Tentativa de desligar a política
- **WHEN** o papel do aplicativo tenta desabilitar a política, alterar tabela ou truncar dados
- **THEN** o banco recusa a operação

### Requirement: Toda tabela com dados de empresa tem política
O sistema SHALL falhar a verificação automática quando existir tabela com identificador de empresa sem política de isolamento ativa.

#### Scenario: Tabela nova sem política
- **WHEN** um desenvolvedor adiciona uma tabela com identificador de empresa e esquece a política
- **THEN** o teste de catálogo falha antes do merge

### Requirement: A transação da requisição não envenena o fluxo
O sistema SHALL tolerar erros esperados dentro de uma requisição (como violação de unicidade tratada) sem abortar o restante do trabalho.

#### Scenario: Duplicata tratada
- **WHEN** um fluxo trata uma violação de unicidade como "já existe" e continua
- **THEN** as consultas seguintes da mesma requisição funcionam normalmente

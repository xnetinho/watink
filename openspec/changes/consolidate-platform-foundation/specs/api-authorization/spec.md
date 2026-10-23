# Spec Delta

## Purpose

Garante que toda ação protegida da API exija uma permissão declarada, impedindo que um usuário autenticado execute o que o seu cargo não permite.

## ADDED Requirements

### Requirement: Toda rota protegida declara a permissão
O sistema SHALL exigir permissão explícita em toda rota autenticada que altera dados, e listar em um registro versionado as raras rotas isentas, cada uma com justificativa.

#### Scenario: Rota nova sem permissão
- **WHEN** um desenvolvedor adiciona uma rota autenticada de escrita sem permissão nem isenção registrada
- **THEN** a verificação automática falha

### Requirement: Ações destrutivas em massa exigem permissão específica
O sistema SHALL exigir permissão própria para excluir contatos em massa, excluir todos os contatos e remover fluxos, funis e filas.

#### Scenario: Atendente sem permissão
- **WHEN** um atendente sem a permissão tenta excluir todos os contatos
- **THEN** a ação é negada com 403 e nada é alterado

#### Scenario: Gerente com permissão
- **WHEN** um usuário com a permissão executa a mesma ação
- **THEN** ela é concluída

### Requirement: Enviar mensagem exige permissão
O sistema SHALL exigir permissão para enviar, reagir, baixar mídia e transcrever mensagens de um ticket, respeitando a visibilidade do ticket.

#### Scenario: Ticket de outro atendente
- **WHEN** o usuário tenta enviar mensagem em um ticket que não pode ver
- **THEN** a ação é negada

### Requirement: A negação não revela existência
O sistema SHALL responder da mesma forma a recurso inexistente e a recurso de outra empresa.

#### Scenario: Identificador de outra empresa
- **WHEN** o usuário pede um recurso que pertence a outra empresa
- **THEN** a resposta é idêntica à de recurso inexistente

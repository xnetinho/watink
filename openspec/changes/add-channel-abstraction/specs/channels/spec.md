# Spec Delta

## Purpose

Permite que o Watink receba e envie mensagens por vários canais (WhatsApp, Telegram, Mercado Livre, OLX, Instagram,
TikTok) sem que o núcleo (tickets, contatos, mensagens, filas, fluxos, permissões) dependa de qual canal é.

## ADDED Requirements

### Requirement: Canal como adaptador
O sistema SHALL tratar cada canal como um adaptador que implementa receber, enviar e conectar, e SHALL declarar o que
o canal permite (capacidades) em dado, não em código espalhado.

#### Scenario: Canal novo não altera o núcleo
- **WHEN** um novo canal é registrado
- **THEN** tickets, contatos, mensagens, filas, fluxos e permissões funcionam com ele sem alteração no núcleo

#### Scenario: Capacidade ausente
- **WHEN** um canal não declara uma capacidade (por exemplo, grupos)
- **THEN** a interface e a API não oferecem essa ação para conversas desse canal

### Requirement: Identidade do contato por canal
O sistema SHALL identificar o contato em cada canal por `(canal, identificador externo)`, e SHALL NOT exigir telefone.

#### Scenario: Canal sem telefone
- **WHEN** chega uma mensagem de um canal que não informa telefone
- **THEN** o contato é criado ou encontrado pelo identificador do canal

#### Scenario: Mesma pessoa em dois canais
- **WHEN** a mesma pessoa escreve por dois canais
- **THEN** o sistema mantém duas identidades e NÃO as une automaticamente

### Requirement: Uma conversa por chave de conversa
O sistema SHALL abrir no máximo um ticket aberto por `(conexão, chave de conversa)`, onde a chave é o contato na maioria
dos canais, o pedido no Mercado Livre e o anúncio na OLX.

#### Scenario: Recompra no Mercado Livre
- **WHEN** o mesmo comprador escreve sobre um segundo pedido
- **THEN** um novo ticket é aberto para esse pedido e o primeiro não é afetado

### Requirement: Regra de resposta aplicada pelo servidor
O sistema SHALL aplicar no servidor a regra de resposta do canal (quem pode iniciar, janela de tempo, limite de texto) e
SHALL devolver o motivo ao cliente.

#### Scenario: Janela expirada
- **WHEN** o atendente tenta responder após a janela do canal
- **THEN** o envio é recusado com um código de erro e o motivo, e o composer já aparece desabilitado com o motivo

#### Scenario: Canal que não permite iniciar
- **WHEN** o atendente tenta abrir conversa nova em um canal que só permite responder
- **THEN** a ação não é oferecida e a API recusa

### Requirement: Entrada única, idempotente e autenticada
O sistema SHALL receber mensagens de todos os canais por um único contrato, deduplicar por
`(conexão, identificador externo da mensagem)` e autenticar cada webhook por segredo da conexão.

#### Scenario: Entrega repetida
- **WHEN** o mesmo evento é entregue duas vezes
- **THEN** uma única mensagem é registrada

#### Scenario: Assinatura inválida
- **WHEN** um webhook chega com assinatura inválida
- **THEN** é recusado sem revelar se a conexão existe

### Requirement: Credenciais cifradas
O sistema SHALL guardar credenciais de canal cifradas em repouso, nunca devolvê-las em resposta nem registrá-las em log.

#### Scenario: Listagem de conexões
- **WHEN** a API lista conexões
- **THEN** nenhuma credencial aparece na resposta

### Requirement: O WhatsApp não regride
O sistema SHALL manter o comportamento do WhatsApp inalterado durante e depois da introdução da abstração.

#### Scenario: Paridade
- **WHEN** a abstração é ativada
- **THEN** receber, enviar, ack, citação e mídia do WhatsApp produzem os mesmos resultados de antes

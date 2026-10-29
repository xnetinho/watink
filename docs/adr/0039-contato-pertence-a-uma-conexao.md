# ADR 0039 — Todo contato pertence a uma conexão

**Status:** Accepted
**Data:** 2026-10-28

## Contexto
`Contacts` era único por empresa e número. Com duas conexões WhatsApp (ou WhatsApp e Telegram) o contato não sabia de qual
conexão derivava, então:
- a importação pegava a **primeira conexão conectada**, sem o usuário escolher;
- o mesmo telefone visto por duas conexões virava um só contato, misturando históricos;
- encaminhar para um contato exigia adivinhar por qual conexão enviar.

## Decisão
1. `Contacts.connectionSource` (`whatsapp` = tabela `Whatsapps`; `channel` = `ChannelConnections`) + `Contacts.connectionId`,
   **obrigatórios**. O par identifica a conexão porque os ids repetem entre as duas tabelas.
2. A unicidade passa a ser por **(empresa, conexão, número)** e **(empresa, conexão, lid)**. O mesmo telefone em duas
   conexões são dois contatos. `ContactIdentities` também ganha `connectionId`.
3. Todo ponto que cria contato recebe a conexão: mensagem recebida, chamada, importação, canal agnóstico e cadastro na tela.
   A API recusa (`422 connection_required`) criar contato sem conexão válida **da mesma empresa**
   (`resolveConnection` é a fonte única dessa regra).
4. **Importar é de UMA conexão**: `POST /contacts/import {connectionId}`; o botão global saiu da tela de Contatos e virou
   ação do menu de cada conexão WhatsApp conectada. Evento `contact.import` sem `sessionId` é recusado.
5. **Encaminhar tem o contato como destino**; a mensagem sai pela conexão do próprio contato, e a conversa é aberta se não
   existir. Entre canais diferentes é recusado. O selo "Encaminhada" exige a mesma conexão de origem e destino (o Telegram só
   encaminha o que o próprio bot enxerga); a cópia sem selo funciona entre conexões do mesmo canal.
6. **Contato WhatsApp criado no Watink é gravado na agenda do celular**, como faz o WhatsApp Web: comando `contact.save` →
   mutação `ContactAction` no patch `critical_unblock_low` (índice `contact`). O resultado volta em `contact.saved` e fica em
   `Contacts.phoneSync` (`pending` | `saved` | `failed`). O contato existe no Watink em qualquer caso; falha não o desfaz.

## Consequências
- Sem migração: o app não tem produção nem dado legado (CLAUDE.md). O banco de desenvolvimento é recriado.
- Removido o backfill de `ContactIdentities` (preenchia identidade de contatos antigos; sem legado, não há o que preencher).
- Contato de grupo, `lid` e número continuam como antes, só que dentro da conexão.
- Dívida conhecida: `source` do WhatsApp só some quando `Whatsapps` migrar para `ChannelConnections` (ADR 0038).

## Não verificado
A gravação na agenda do celular segue o formato que o `whatsmeow` decodifica de volta (`ContactAction`), mas **não foi
comprovada com um aparelho real**: o resultado depende de o WhatsApp aceitar a mutação do dispositivo vinculado.

# ADR 0038: Identificação do ticket por canal e conexão

## Status
Aceito

## Contexto
O ticket do Telegram mostrava um selo de canal; o do WhatsApp não mostrava nada, e nenhum dos dois dizia **qual conexão** o atendia. Com várias conexões (dois números de WhatsApp, dois bots) o operador não sabia por onde estava respondendo. Os próximos canais (Mercado Livre, OLX, Instagram, TikTok) têm o mesmo problema.

## Decisão
1. **Contrato único, neutro ao canal.** Todo ticket sai da API com `channel` (o tipo) e `connection`:
   ```json
   { "channel": "telegram", "connection": { "kind": "telegram", "id": 2, "name": "Jarvis", "source": "channel" } }
   ```
   - `kind`: chave do catálogo de canais (`domain.ChannelKind` no backend, `config/channels.ts` no frontend).
   - `id` / `name`: a conexão dentro da empresa.
   - `source`: tabela do `id` (`whatsapp` = `Whatsapps`; `channel` = `ChannelConnections`). Existe porque as duas ainda coexistem e **some** quando o WhatsApp migrar para `ChannelConnections`.
   - Conexão apagada: `connection` ausente; a tela mostra só o nome do canal, nunca um nome inventado.
2. **O backend resolve, a tela só desenha.** A regra "de que tabela vem a conexão" mora em um lugar: `services.ConnectionOf` (um ticket) e `resolveTicketConnections` (lote, no máximo 2 consultas, usado pela lista). O frontend não tem `if` por canal.
3. **Também no tempo real.** O ticket que nasce por mensagem só chega à lista pelo evento SSE; os dois emissores de mensagem recebida (`event_listener`, `channel_inbound`) passam pelo mesmo enriquecimento (`withConnectionIdentity`), senão ele apareceria sem selo até recarregar.
4. **Um componente, três lugares.** `TicketConnectionBadge` (ícone do canal + nome da conexão) na lista de tickets, no cabeçalho da conversa e no painel de detalhes. Cor **só por canal** (do catálogo); conexões do mesmo canal se distinguem pelo nome.

## Como adicionar um canal novo (Mercado Livre, OLX, Instagram, TikTok...)
1. Backend: adaptador em `domain.ChannelRegistry` e `ChannelConnection` com `Kind` e `Name` (ADR 0033/0037). **Nada** muda na identificação do ticket.
2. Frontend: uma entrada em `CHANNELS_CATALOG` (nome, cores) e o ícone em `channelIcon`. Canal ainda fora do catálogo não quebra: cai no genérico (ícone de globo) com o nome da conexão.

## Consequências
- Positivas: um contrato para todos os canais; WhatsApp ganha identificação; duas conexões do mesmo canal se distinguem.
- Custo: uma consulta extra por lote na listagem e uma por mensagem recebida no evento; ambas por chave primária e só duas colunas.
- Dívida declarada: `source` e a bifurcação `Whatsapps` x `ChannelConnections` existem até a migração do WhatsApp (tarefas 3.x do OpenSpec, ainda abertas).

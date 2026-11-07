# Fatia 5: contrato entre backend e frontend

Escrito em 7/nov/2026. É o único acordo que os PRs da fatia 5 compartilham; cada PR muda UMA área e só pode depender do que está
aqui. Meta final (fatia 6): remover `Whatsapps`, as rotas `/whatsapp/*` e o par (origem, id).

## 1. Rotas

A conexão, de qualquer canal, vive em `/channels/connections`. As rotas antigas continuam respondendo até a fatia 6, **com o mesmo
corpo**, para que um PR de frontend possa migrar sem esperar os outros.

| Hoje | Passa a ser | Corpo/resposta |
|---|---|---|
| `GET /whatsapp` | `GET /channels/connections` | lista **todos** os canais (hoje só os não-WhatsApp) |
| `GET /whatsapp/:id` | `GET /channels/connections/:id` | detalhe |
| `POST /whatsapp` | `POST /channels/connections` com `kind:"whatsapp"` | mesmo corpo do WhatsApp (sem `token`) |
| `PUT /whatsapp/:id` | `PUT /channels/connections/:id` | atualização parcial |
| `DELETE /whatsapp/:id` | `DELETE /channels/connections/:id` | |
| `GET /whatsapp/:id/stats` | `GET /channels/connections/:id/stats` | |
| `PUT /whatsapp/:id/keepalive` | `PUT /channels/connections/:id/keepalive` | |
| `PUT /whatsapp/:id/presence-policy` | `PUT /channels/connections/:id/presence-policy` | |
| `POST/PUT /whatsappsession/:id` | `POST /channels/connections/:id/session` | inicia sessão |
| `DELETE /whatsappsession/:id` | `DELETE /channels/connections/:id/session` | encerra sessão |
| `POST /whatsappsession/all` | `POST /channels/connections/session/restart-all` | |

As ações específicas do WhatsApp (`stats`, `keepalive`, `presence-policy`, `session`) devolvem **422** `not_supported_by_channel`
quando a conexão é de outro canal.

## 2. Forma da conexão (resposta)

Uma lista só, objetos homogêneos. Campos comuns a todo canal: `id`, `kind`, `name`, `status`, `isDefault`, `greetingMessage`,
`farewellMessage`, `identifier` (número/@usuário), `displayName`, `avatarUrl`, `firstConnectedAt`, `lastConnectedAt`, `lastError`,
`queues`. Campos só do WhatsApp (presentes quando `kind == "whatsapp"`): `keepAlive`, `presencePolicy`, `presenceIdleMinutes`,
`engineType`, `syncHistory`, `syncPeriod`, `proxyMode`, `proxyId`, `proxyGroupId`, `connectionGroupId`, `lastRiskCode`,
`lastRiskAction`, `lastRiskMessage`, `lastRiskAt`, `wid`.

**Aliases temporários (somem na fatia 6):** a resposta também carrega `number` (= `identifier`), `profilePicUrl` (= `avatarUrl`) e
`firstConnection` (= `firstConnectedAt`), para o frontend antigo não quebrar durante a transição.

## 3. Chave de conexão nos objetos de domínio

Todo objeto que aponta para uma conexão (ticket, fluxo, assistente, chamada, campanha, usuário, evento de chamada) expõe e aceita
**`connectionId`** (inteiro). `whatsappId` **deixou de existir** no JSON, nas queries (`/groups?connectionId=`) e nos corpos; não
há alias. Os caminhos `/whatsapp/*` e `/whatsappsession/*` (parâmetro de rota `:whatsappId`) seguem só até a fatia 6; o frontend
migrado não os usa. Rota do webchat público segue com o parâmetro de rota `:whatsappId` (só nome de parâmetro).

## 4. O par (origem, id) deixa de existir

Ids de conexão são únicos em `ChannelConnections` (o `Whatsapps` não é mais tabela). Somem do JSON e das queries:
`TicketConnectionInfo.source`, `connectionSource` do contato (corpo, resposta e filtro `?connectionSource=`). Uma conexão é
identificada só por `id` e o canal vem de `kind`. No frontend, `lib/connections/connectionRef.ts` perde `source`, `connectionKey`
vira `String(id)` e `parseConnectionKey` vira `Number(key)`; `ConnectionOption = {id, kind, name, status}`.
Filtro de contatos: `?connectionId=<id>` (sem origem).

## 5. Eventos em tempo real

`whatsappSession` e `channelConnection` já carregam a mesma carga. O frontend migrado escuta só `channelConnection`; o evento
`whatsappSession` continua sendo emitido até a fatia 6.

## 6. Ordem (um PR só)

Como `whatsappId` sumiu do contrato, backend e frontend precisam entrar juntos: um único PR (`feat/migracao-fatia-5`), com as áreas abaixo
feitas em paralelo por agentes diferentes (arquivos disjuntos).

## 7. Áreas (histórico do plano inicial)

| PR | Dono | Depende de |
|---|---|---|
| 5a | backend: rotas e forma da seção 1 a 3 | — |
| 5b | frontend: tela de Conexões, modais de conexão, sessão/QR, hooks de conexão | 5a |
| 5c | frontend: Tickets, atendimento, transferência, `types/Ticket.ts` | 5a |
| 5d | frontend: FlowBuilder, Assistentes, Webchat, Usuários, Acessos | 5a |
| 5e | frontend: Grupos, Comunidades, Campanhas, Chamadas | 5a |
| 6 | backend + frontend: remover rotas e tabela antigas, aliases, recriar o `lt` | 5a a 5e |

5b a 5e tocam arquivos diferentes e rodam em paralelo depois do 5a. Cada um deixa `tsc`, `eslint --max-warnings 0` e `vitest`
verdes, e tem busca de `whatsappId` e de `/whatsapp` zerada na sua área.

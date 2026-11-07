# Migração `Whatsapps` → `ChannelConnections` (item E): auditoria e plano de fatias

Escrito em 5/nov/2026. **Nada foi alterado no código.** Esta é uma auditoria do estado real e uma proposta de fatias para
aprovação. O plano de origem é `openspec/changes/consolidate-platform-foundation/tasks.md` (Onda 3 e Onda 4).

## 1. O que é

Hoje o Watink tem **duas tabelas de conexão** convivendo:

| | `Whatsapps` (antiga) | `ChannelConnections` (agnóstica) |
|---|---|---|
| Quem usa | todas as conexões **WhatsApp** | **Telegram** e os canais futuros |
| Campos | tudo do WhatsApp: QR, número, proxy, presença, risco, `engineType`, `wid` | genéricos: `kind`, credenciais cifradas, `config` JSON |
| Rotas | `/whatsapp/*` (13 rotas; a tela de Conexões usa estas) | `/channels/connections/*` e `/connections/*` |
| Tabela de detalhes | os campos estão na própria linha | `whatsapp_connection_details` (1:1, **existe e está vazia**) |

A migração é fazer do WhatsApp **mais uma linha de `ChannelConnections`** (`kind='whatsapp'`), com os campos próprios em
`whatsapp_connection_details`, e eliminar `Whatsapps`, `whatsapp_queues` e a coluna `whatsappId` das tabelas dependentes.

## 2. Auditoria: o plano marca como concluído o que o código não fez

As tarefas 3.3, 3.4 e 3.7 estão `[x]` no `tasks.md`. Conferido contra o código e o banco do `lt` em 5/nov/2026:

| Tarefa | O plano diz | O que os PRs fizeram de fato | Estado real hoje |
|---|---|---|---|
| **3.2** `whatsapp_connection_details` 1:1 | concluída | PR #39 criou a tabela e o modelo | **Existe, mas nada escreve nela** (0 linhas; nenhum código de produção usa `WhatsAppConnectionDetails`) |
| **3.3** remover `Tickets.whatsappId` e o sentinela `0` | concluída | PR #40 só acrescentou `constraint:OnDelete:CASCADE` e `references:ID` na FK de `ChannelConnectionID` | **`Ticket.WhatsappID` existe** (`int`, não nulo na prática). No `lt`: 6 tickets com `whatsappId>0` e `channelConnectionId` nulo |
| **3.4** remover `whatsappId` de 8 tabelas | concluída | PR #41 **acrescentou** `ChannelConnectionID` ao lado e pôs `omitempty` no JSON de `whatsappId` | **As 9 tabelas ainda têm `whatsappId`** (Assistants, CallLogs, Flows, Tickets, Users, group_caches, group_campaigns, group_campaign_targets, group_campaign_sends). Nenhuma teve a coluna removida |
| **3.7** remover `Whatsapps`, `whatsapp_queues` e `/whatsapp/*` | concluída | PR #45 alterou **uma linha** do `tasks.md` (o `[ ]` virou `[x]`), sem tocar em código | **A tabela, `whatsapp_queues` e as 13 rotas existem.** 37 arquivos de produção usam `models.Whatsapp` |
| **4.5** adaptador WhatsApp real | concluída | existe `channels/whatsapp/adapter.go` (463 linhas), registrado em `routes.go:142` | **Registrado, mas fora do caminho real**: nenhum ticket WhatsApp tem `channelConnectionId`, então o envio nunca o usa (cai no ramo legado de `message_deliver.go`) |

O texto das tarefas 3.3, 3.4 e 3.7 diz "verificar que a busca por `whatsappId` no código de produção retorna zero" e "verificar
que nenhum SQL ou rota de produção os referencia". Essa verificação **não foi feita**: a busca retorna 62 arquivos de produção.

**Recomendação:** reabrir as tarefas 3.2 (parte de escrita), 3.3, 3.4, 3.7 e 4.5 (ligação ao caminho real) no `tasks.md`, e só
marcá-las de novo quando a verificação descrita em cada uma passar.

## 3. Volume real (produção / teste)

| O que | Produção | Teste |
|---|---|---|
| arquivos com `models.Whatsapp` | 37 | 56 |
| arquivos com `WhatsappID`/`WhatsAppID` | 62 | 46 |
| arquivos com a tabela `"Whatsapps"` em SQL | 22 | 6 |
| arquivos com `whatsapp_queues` | 10 | - |
| frontend: arquivos com `whatsappId` | 65 | - |
| frontend: arquivos que usam `/whatsapp` ou `useWhatsAppsQuery` | 16 | - |

`WhatsappID` em produção, por camada: plugins 20, controllers 19, models 10, flow 4, calls 4, infrastructure 2, services 1,
domain 1, application 1.

## 4. O que torna isto difícil (fatos que moldam as fatias)

1. **O engine identifica a sessão por um inteiro**, na routing key `wbot.<tenant>.<id>.<comando>`. O engine não conhece tabela
   nenhuma; trocar de tabela só muda **qual número o business manda**. Mas **a sessão do whatsmeow está amarrada a esse número**
   no banco do engine (por exemplo `558387453500:6` ↔ id 1). Mudar o id de uma conexão pareada exige **parear de novo**.
2. **Os ids colidem entre as tabelas.** No `lt`: `Whatsapps` = {1 Teste, 2 iZap}; `ChannelConnections` = {1 Telegram}. O ticket
   do WhatsApp com `whatsappId=1` não pode receber `channelConnectionId=1` (é o Telegram). Unificar exige decidir o destino dos ids.
3. **Duas camadas de adaptador coexistem.** O envio usa `ticket.ChannelConnectionID != nil` para escolher o caminho: com valor,
   segue o adaptador agnóstico (Telegram); sem, segue o caminho WhatsApp direto por RabbitMQ. Os recursos feitos nesta frente
   (visualização única, figurinha, GIF) vivem **só no caminho legado** e hoje **recusam** o canal agnóstico (`422`). Migrar o
   WhatsApp para o caminho agnóstico exige antes levar esses recursos para o adaptador.
4. **O adaptador WhatsApp reconstrói um `models.Whatsapp` falso** a partir da `ChannelConnection` (`Connect` e `Disconnect`):
   ele mesmo ainda depende do modelo antigo.
5. **Os campos próprios do WhatsApp (24 deles)** não existem em `ChannelConnection`: QR, número, bateria, proxy, presença,
   `keepAlive`, risco, `engineType`, `wid` e outros. Parte já tem casa em `whatsapp_connection_details` (proxy, sync,
   `keepAlive`, `wid`, risco); o resto (QR, número, foto, bateria, presença, `engineType`) **não tem** e precisa de decisão.
6. **O frontend usa `/whatsapp/*`** em 16 arquivos e `whatsappId` em 65.

## 5. Impacto no resto do backlog

| Item | Impacto da E |
|---|---|
| **G.** múltiplas bibliotecas WhatsApp | **Depende da E.** `EngineType` mora em `Whatsapp`; a escolha por conexão precisa estar na tabela unificada |
| **F.** voz | **Toca a E.** `CallLogs` e `Flows` têm `whatsappId` |
| **H.** mídia em S3 por tenant | pouco (o limite é por tenant, não por conexão) |
| **I.** retenção por tenant | pouco, mas a retenção apaga `Tickets`, e a E muda a chave de conexão dos tickets |
| figurinhas, versão do WhatsApp, GIF | nenhum (são por empresa/instalação) |

## 6. Plano de fatias proposto

Regras do projeto: sem produção nem legado, **sem migração incremental e sem coexistência**, coluna ou tabela nova só no
modelo GORM/baseline (no `lt`, criar à mão com o DDL idêntico), suíte inteira verde em cada fatia, um PR por fatia.

**Decisões que precisam do dono antes da fatia 1 (não decidi):**

- **D1. Destino dos ids.** (a) o `lt` é recriado do zero e o WhatsApp ganha ids novos em `ChannelConnections` (as conexões
  precisam **parear de novo**); ou (b) as linhas de `Whatsapps` são copiadas para `ChannelConnections` **mantendo o id** e o id do
  Telegram muda para um valor livre. A (b) preserva as sessões do whatsmeow; a (a) é mais simples e honesta com "sem legado".
- **D2. Onde moram QR, número, foto, bateria, presença e `engineType`.** Em `whatsapp_connection_details` (mais colunas), em
  `ChannelConnection.config` (JSON) ou em colunas novas de `ChannelConnections`.
- **D3. Os recursos de mídia (visualização única, figurinha, GIF) passam para o adaptador agnóstico ou o WhatsApp mantém o
  caminho próprio?** O backlog G (outras bibliotecas) favorece o adaptador.

**Fatias (cada uma verde de ponta a ponta, em ordem):**

| # | Fatia | Verificação que fecha a fatia |
|---|---|---|
| 0 | **Corrigir o `tasks.md`**: reabrir 3.2, 3.3, 3.4, 3.7 e 4.5 e registrar esta auditoria | PR só de documentação |
| 1 | **Detalhes escritos de fato**: toda criação/edição de conexão WhatsApp grava `whatsapp_connection_details` (e os campos de D2); leitura passa a vir dali | teste de repositório: conexão WhatsApp grava e lê seus detalhes; nenhum código lê esses campos de `Whatsapps` |
| 2 | **Tickets**: `channelConnectionId` obrigatório, `whatsappId` removido de `Tickets` (e o sentinela `0`) | teste: ticket de qualquer canal grava e consulta por `connectionId`; busca por `Ticket.WhatsappID` = 0 |
| 3 | **Dependentes, um PR por grupo**: `Users`, `Flows`, `Assistants`, `CallLogs`, `group_*` (3.4) | a busca por `whatsappId` no código de produção cai a zero a cada PR |
| 4 | **Envio e recebimento pelo adaptador agnóstico** (depende de D3): levar visualização única, figurinha e GIF para o adaptador | os testes de recurso passam contra o adaptador; o ramo legado de `message_deliver.go` deixa de ser alcançável |
| 5 | **Frontend**: tela de Conexões e demais telas passam a `/channels/connections`; `whatsappId` vira `connectionId` nos tipos | `tsc`, `eslint`, vitest; busca por `/whatsapp/` e `whatsappId` no frontend = 0 |
| 6 | **Remover `Whatsapps`, `whatsapp_queues` e `/whatsapp/*`** (3.7) | busca de produção por `models.Whatsapp`, `"Whatsapps"` e `whatsapp_queues` = 0; `lt` reconstruído e conexões pareadas funcionando |

Ordem alternativa: a fatia 4 pode vir antes da 2 se D3 for "adaptador agnóstico", porque tirar `whatsappId` dos tickets só faz
sentido quando o envio já não o usa.

## 7. Riscos

- **Alto risco de regressão**: 37 + 62 arquivos de produção e ~100 de teste tocados, em um caminho (envio e recebimento de
  mensagem) que é o coração do produto.
- **Sessões do whatsmeow**: dependendo de D1, todas as conexões pareadas precisam ser lidas de novo por QR.
- **Recursos recentes** (visualização única, figurinha, GIF) só existem no caminho legado; a fatia 4 pode regredi-los.
- **O plano marcou como feito o que não estava**: antes de confiar em qualquer outra tarefa `[x]` do `tasks.md`, vale uma
  verificação por amostragem.

## 8. Decisões do dono (5/nov/2026) e plano revisado

**D1. Destino dos ids: repareamento total.** O `lt` é recriado do zero; as conexões WhatsApp ganham ids novos em
`ChannelConnections` e as duas precisam ler o QR de novo. Sem preocupação com legado, sem cópia de linhas, sem colisão de ids.

**D2. Onde moram os campos da conexão.** Classificados pelo uso real no código:

| Grupo | Campos | Destino |
|---|---|---|
| A. Identidade da conta (todo canal tem) | identificador (número no WhatsApp, `@usuario` no Telegram), nome de exibição, foto | **colunas de `ChannelConnections`**: `identifier`, `displayName`, `avatarUrl` |
| B. Estado da sessão (todo canal tem) | `status`, `lastError`, `isDefault` (já são colunas), `firstConnectedAt`, `lastConnectedAt` | **colunas de `ChannelConnections`** (acrescentar as duas datas) |
| C. Específico do WhatsApp/biblioteca | `engineType`, `wid`, `keepAlive`, `syncHistory`, `syncPeriod`, proxy (3), risco (4), `presencePolicy`, `presenceIdleMinutes`, `connectionGroupId` | **`whatsapp_connection_details`** (1:1, já existe) |
| D. Mortos ou temporários | `battery`, `plugged`, `retries`, `session`, `qrcode` | **eliminar**: o engine nunca envia bateria (todos vazios no lt) e o QR já vai por evento SSE, não pelo banco |

`engineType` mora em `whatsapp_connection_details` para que o item G (outras bibliotecas) encaixe sem tocar em
`ChannelConnections`. O JSON `config` fica só para o que é particular de um canal (modo polling/webhook do Telegram): número e foto
não vão nele porque não têm tipo nem índice, a tela os lê direto e a regra "o mesmo número não está em duas conexões" exige coluna.

**D3. Todos os recursos passam pelo adaptador agnóstico.** Visualização única, figurinha, GIF, biblioteca de figurinhas, edição,
recibos. Hoje eles recusam o canal agnóstico com 422; passam a viver no adaptador.

**Plano revisado** (a fatia 3 passa a ser pré-requisito de tirar `whatsappId` dos tickets):

| # | Fatia |
|---|---|
| 0 | Corrigir o `tasks.md` (reabrir 3.2, 3.3, 3.4, 3.7, 4.5) e registrar estas decisões |
| 1 | Esquema novo: colunas agnósticas em `ChannelConnections`, `engineType` e demais em `whatsapp_connection_details`, remover os 5 campos mortos |
| 2 | Conexão WhatsApp criada, editada e lida pela API agnóstica (`/channels/connections`) |
| 3 | Recursos de mídia no adaptador agnóstico |
| 4 | Tickets e dependentes trocam `whatsappId` por `connectionId` |
| 5 | Frontend |
| 6 | Remover `Whatsapps`, `whatsapp_queues`, `/whatsapp/*`; recriar o `lt` e reparear |

## 9. Andamento (7/nov/2026)

**Fatias 0 a 3 entregues** (#150 a #159): auditoria, esquema novo, conexão WhatsApp em `ChannelConnections` +
`whatsapp_connection_details`, e envio, operações, mídia, contatos, resposta rápida, flow, plugins, chamadas (comandos **e** fluxo
de áudio/vídeo) pelo adaptador do canal. O business não publica mais comando de WhatsApp direto no RabbitMQ pelos controllers.
Validado em celular real: texto, citação, visualização única, figurinha, GIF, voz, enquete, reação, voto e **chamada de voz recebida**.

**Achado recorrente da fatia 3:** cinco defeitos de escopo de empresa (RLS) que os testes comuns não pegavam, porque o banco de
teste não aplica o RLS por papel. Cada um ganhou teste com RLS real (`internal/database/rls_session_test.go`). Ao mexer em
leitura de conexão, ticket ou chamada, o teste novo deve rodar com RLS real.

**Fatia 4, decisão D4 (dono, 7/nov/2026): o ticket do WhatsApp ganha `conversationKey` (opção B).** Hoje o ticket do WhatsApp se
identifica por contato + `whatsappId` e não tem `conversationKey`; o do Telegram por `channelConnectionId` + `conversationKey`, com o
índice único `idx_tickets_open_conversation`. Depois da migração os dois usam o mesmo caminho: o WhatsApp passa a gravar
`conversationKey` (o JID do chat), entra no índice único e na mesma busca.

**Ordem da fatia 4, um PR cada:**

| PR | Conteúdo | Estado |
|---|---|---|
| 4a | `Flows`, `Users`, `Assistants`, `CallLogs`: `whatsappId` sai, fica só `channelConnectionId` (a chave JSON segue `whatsappId` até a fatia 5) | em andamento |
| 4b | `group_caches`, `group_campaigns`, `group_campaign_targets`, `group_campaign_sends` | a fazer |
| 4c | `Tickets`: `channelConnectionId` obrigatório, `conversationKey` do WhatsApp, índice único, busca unificada; sai `Ticket.WhatsappID` e o sentinela `0` | a fazer |

**Banco já migrado (`lt`):** coluna removida do modelo fica órfã até o banco ser recriado (fatia 6). Não há migração incremental.


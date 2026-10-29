# Contrato frontend ↔ backend: rota que a tela chama e o backend não tem

## O bug (27/out/2026): "connection not found" ao clicar em Ver QRCode
- Sintoma: o modal/tela de detalhe da conexão WhatsApp respondia `connection not found`.
- Causa: na tarefa 7.1 o `useConnectionFetch` passou a chamar `GET /connections/:id`, que lê **`ChannelConnections`**. As conexões WhatsApp continuam em **`Whatsapps`** e só são servidas por `GET /whatsapp/:id`. O mesmo erro de troca estava em 4 telas (Grupos, FlowManager, FlowBuilder, AssistantCreator), que listavam via `/connections` e ficavam **vazias** para quem só tem WhatsApp.
- Correção: essas chamadas voltaram a `/whatsapp`. A listagem de Conexões já mescla as duas fontes (`useConnections`).

## Por que passou
1. **Migração declarada pronta sem estar.** As tarefas 3.2, 3.3, 3.4, 3.7 e 7.1 estão `[x]`, mas o código de produção ainda usa `Whatsapps` (tabela, 8 rotas `/whatsapp/*`, `Tickets.whatsappId` e mais 8 tabelas com `whatsappId`). O frontend foi migrado para o modelo novo **antes** de o backend existir nele.
2. **Teste de componente com mock.** Os testes mockam `api.get`: provam que a tela renderiza, não que a rota existe.
3. **Nenhuma checagem cruzada** entre o que o frontend chama e o que o router registra.

## O que impede de voltar
`business/internal/routes/frontend_contract_test.go` (`TestFrontendCallsExistOnBackend`) lê as chamadas literais `api.<verbo>("/rota")` do frontend e exige que cada uma case com uma rota registrada em `SetupRoutes` (método + caminho, com `:param` e `${...}` casando por segmento). Roda no CI.

- Falha ao reintroduzir um `api.get('/connections/${id}')` onde a rota não existe? **Sim, quando a rota realmente não existir.** Atenção: hoje `GET /connections/:id` **existe** (registrada na 7.1), então esse teste **não** teria pego o bug original. Ele pega rota inexistente, não rota que existe e lê a tabela errada. Para esse caso vale o item abaixo.
- Dívida antiga fica em `frontendRouteDebt`, com motivo. O teste falha se uma entrada deixar de ser verdadeira (obriga a limpar).

## Regras para o próximo canal e para migrações
1. **Uma rota nova só vale se o dado que ela serve já migrou.** Antes de apontar a tela para `/connections`, o backend precisa devolver o WhatsApp por ela (hoje não devolve). Migração é backend → dados → frontend, nunca o inverso.
2. **Não marque a tarefa `[x]` por existir o código novo.** Critério é o antigo ter sido removido: `Whatsapps` sem referência, rotas antigas fora do router.
3. **Teste a tela contra o backend real** (smoke com API real) para os fluxos críticos: conectar, ver QR, enviar. Mock não prova contrato.
4. **Resposta por canal deve ter o mesmo formato.** `GET /whatsapp/:id` e `GET /channels/connections/:id` devolvem formatos diferentes (um tem `qrcode`, `proxy`, `queues`; o outro tem `config`, `lastError`). Antes de unificar a rota, unifique o DTO.
5. **Todo canal novo registra seu `GET /<canal>/:id` e entra no mesmo teste de contrato.**

## Dívida conhecida (não é regressão; já era assim antes da consolidação)
Chamadas do frontend sem rota no backend, hoje documentadas em `frontendRouteDebt`: `GET /tickets/:id/participants` (menção em grupo), `GET /tags/:id` (TagModal), `GET /v1/plugins/installed` (prefixo `/v1` duplicado), `DELETE /messages/:id`, `POST /tickets/:id/history`, `PUT /users/:id/configs`, `POST /auth/reset-password`, rotas do painel SaaS e `register/*` (só existem com `SAAS_*` configurado).

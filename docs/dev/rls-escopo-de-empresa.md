# RLS e escopo de empresa: armadilha do `db` global

Leitura obrigatória para quem escreve serviço, adaptador de canal ou goroutine nova. Base: ADR 0034 (RLS real).

## Regra
O handle global `*gorm.DB` é um `tenantdb.Router`. Cada comando roda conforme o escopo que vem no `context`:

| ctx | O que acontece |
|---|---|
| escopo de empresa (`tenantdb.TenantCtx` / `auth.GetScoped`) | conexão fixada à empresa; o banco só enxerga as linhas dela |
| escopo de sistema (`tenantdb.SystemCtx`) | pool `BYPASSRLS`; só para varredura global e descoberta de empresa (restrito por teste de arquitetura) |
| **sem escopo** | tabela com `tenantId` é recusada com `ErrNoScope` |

`WHERE "tenantId" = ?` escrito à mão **não basta**: o filtro não substitui o escopo. Sem escopo, a consulta falha, mesmo com o filtro correto.

## O bug (27/out/2026): "Internal server error" ao clicar em Conectar
- Sintoma: toast genérico em `POST /whatsappsession/:id`; o comando nunca chegava ao engine.
- Log: `[tenantdb] SEM ESCOPO em ?: UPDATE "Whatsapps" SET "status"=...` e `tenantdb: consulta sem escopo de empresa...`.
- Causa: `WhatsAppSessionService` e `enginego.Provider` usavam `wss.db.Model(...)` / `p.db.Where(...)` sem `ctx`. O serviço recebia só o modelo `Whatsapp`, não o `ctx` da requisição.
- Por que passou despercebido:
  1. Os testes usam `testutil.NewTestDB`, um Postgres **sem Router e sem RLS**: a consulta sem escopo passa.
  2. O erro virou 500 genérico no toast; a causa só aparece no log.
  3. O `CLAUDE.md` dizia "RLS inerte", o que induzia a confiar no filtro manual.
- Efeitos colaterais do mesmo defeito: o lock `session:start:<id>` vazava (próximo clique dava 409 em vez do erro real); o rollback de status e a resolução de proxy (`composeProxyURL`, `pickGroupProxy`) tinham a mesma falha; `RestartAllSessions` engolia o erro e respondia 200.

## Como evitar (checklist para código novo)
1. **Todo acesso ao banco recebe `ctx` com escopo.** Em handler: `auth.GetScoped(c, "Tabela")` e `db.WithContext(c.Request.Context())`. Em serviço sem requisição: `tenantdb.Do(ctx, db, tenantID, func(tx *gorm.DB) error {...})` ou `TenantCtx` com `defer done()`.
2. **Assinatura de serviço/porta carrega `ctx`** (`StartSession(ctx, ...)`). Nunca `context.Background()` num caminho que toca o banco.
3. **Goroutine longa** (polling, worker, consumidor): abra escopo **por unidade de trabalho** (`tenantdb.WithTenant` por mensagem/evento) e use `tenantdb.Detach` quando herdar o `ctx` da requisição. Escopo fixa uma conexão: não o mantenha aberto durante espera externa (`tenantdb.Release`).
4. **Descoberta da empresa** (webhook que só tem o id da conexão): `SystemCtx` só para achar o `tenantId`, fechar, e abrir `TenantCtx` para o resto. Padrão em `controllers/channel_webhook.go`.
5. **Todo `return` depois de `SetLock` libera o lock** (`DelLock`), inclusive no erro do primeiro `UPDATE`.
6. **Nunca ignore erro de chamada em laço** (`_ = svc.Start(...)`): conte e devolva (ex.: 207).
7. **Teste com RLS real.** Para qualquer serviço que escreve em tabela com `tenantId`, escreva ao menos um teste sobre o banco migrado de verdade (`migratedDB` + `routerOver`, em `internal/database`). Modelo: `rls_session_test.go` (`TestStartWhatsAppSession_ComRLSReal`). `testutil.NewTestDB` **não** prova isolamento.
8. **Erro de escopo é bug, não "falha do servidor"**: ao ver `ErrNoScope` no log, corrija o `ctx`; não relaxe o Router nem use `SystemCtx` para "fazer funcionar".

## Checklist "novo canal" (acrescenta ao ADR 0037)
- [ ] `Connect`/`Disconnect`/`Send` usam o `ctx` recebido; o adaptador **não** guarda nem usa `*gorm.DB` global.
- [ ] Quem chama o adaptador (controller, polling, retomada no boot) abre o escopo da empresa antes.
- [ ] Entrada de mensagem (webhook ou polling) preenche `TenantID` e `ChannelConnectionID` e roda `Receive` dentro de `TenantCtx`.
- [ ] Varredura no boot (retomar polling) usa `SystemCtx` só para listar; cada conexão volta a rodar no próprio escopo.
- [ ] Teste de integração com RLS real cobre conectar, receber e enviar.

## Auditoria do Telegram (27/out/2026)
Não sofre do mesmo defeito. Verificado por leitura e contra o servidor real:
- `telegram/adapter.go` e `client.go`: sem `gorm`; só `ctx` em chamadas HTTP.
- Webhook (`channel_webhook.go`): `SystemCtx` só para descobrir a empresa, depois `TenantCtx` para o resto.
- Polling: o handler (`routes.go`) envolve cada mensagem em `tenantdb.WithTenant`; a retomada no boot lista com `SystemCtx`.
- Criar/salvar/listar/excluir conexão Telegram via API real: sem nenhum `SEM ESCOPO` nos logs.
- Pontos de atenção (não são o mesmo bug): `connectAndSave` ignora o erro do `db.Save`; o polling guarda o `offset` só em memória e depende de deduplicação; `baseURL` padrão do webhook aponta para `homolog.watink.com` quando a requisição não informa.

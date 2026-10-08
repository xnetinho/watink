# Tasks

> Fonte: `proposal.md` e `design.md`. **Regra:** um PR por área, cada lote verificado por build + testes e **sem
> mudar comportamento**. Remoção só com **grafo de imports/`deadcode`/`knip` E confirmação manual**.
> Decisões do dono (out/2026): remover `watink-smtp-go`; **manter** `fetch_url_crawl.go`, documentar e pôr no roadmap;
> remover os PNGs sem uso. Os bugs de log, RLS e source maps são changes separadas (já entregues).

## PR A: repositório e documentação (`chore/purge-repo-dead-config`)

- [x] A.1 Remover `business/web/` (160 arquivos, 15,8 MB, bundle estale; nenhuma referência) e ignorar no `.gitignore`
- [x] A.2 Remover `frontend/frontend/` (3 arquivos vazios versionados por engano)
- [x] A.3 Remover `update.sh`, `business/run_migrate.go` e `plugins/watink-smtp-go/` (esqueleto de 42 linhas, sem uso)
- [x] A.4 Remover `scripts/gen-proto.sh`, `init_db.sh`, `rebuild-backend-with-embed.sh`, `smoke-navigation.js` (sem
      referência). **Ficam** `setup-branch-protection.sh`, `smoke-docker.sh`, `report-duplicate-contacts.sql` (manuais)
- [x] A.5 `package.json` da raiz: remover scripts `windows:*` e `main: ecosystem.config.js`; remover
      `ecosystem.config.js` e a seção PM2 de `docs/dev/commands.md`
- [x] A.6 Workflows: remover `push-image-frontend.yaml` (dispara em `master`, nunca roda). **Ficam**
      `build-frontend.yaml` (avaliar) e `publish-ghcr-fork.yml`
- [x] A.7 `dependabot.yml`: remover entradas de pastas inexistentes (`marketplace-hub`, `plugin-manager`,
      `legacy/backend`, `legacy/engine-standard`) e **adicionar** `engine-go` e `e2e`; `codeql/js-config.yml` e
      `.gitignore`: tirar `legacy`, `marketplace-hub`, `plugin-manager` e os artefatos de Python
- [x] A.8 `CLAUDE.md`: tirar da tabela de Services `Marketplace Hub`, `Backend Node (legacy)` e `Engine Node (legacy)`;
      corrigir a lista de plugins; remover a contradição sobre `marketplace-hub`
- [x] A.9 ADR 0018 `Superseded by 0028`; corrigir o link quebrado de `docs/frontend/chats/OVERVIEW.md`; aviso de
      "histórico" em `docs/legacy-backend/` e `docs/legacy-engine/`; `ESTADO_ORQUESTRATOR.md` para `docs/`
- [x] A.10 PR #4 contra `develop`, CI verde (workflows só se validam na CI), merge

## PR B: frontend (`chore/purge-frontend-dead-code`)

- [x] B.1 Cadeia órfã do chat: `MessageListContainer.tsx`, `MessageItem.tsx`, `MessageItem.spec.tsx`,
      `MessagesList/MessageMedia.tsx`; remover o `declare module '@virtuoso.dev/message-list'` de `global.d.ts`
- [x] B.2 Páginas e modais sem rota: `pages/Tenants/`, `components/TenantModal/`, `components/PermissionTransferList/`
- [x] B.3 Componentes soltos: `ColorPicker`, `InfoCard`, `Title` e `ui/title`, `layout/header`, `ComingSoonItem`,
      `NewTicketModal/index.tsx`, `useTicket`, `useTicketsQuery`, `AdminSectionDivider`, `ConnectionIcon`,
      `FlowBuilder/ContentModal`, `FlowBuilder/StartNodeModal`, `Helpdesk/ProtocolDrawer`, `rules.ts`
- [x] B.4 Dependências sem uso: `react-virtuoso`, `@testing-library/user-event`, `@radix-ui/react-avatar`,
      `react-color`. **Ficam** `express`/`express-rate-limit` (`frontend/server.js`), plugins do ESLint, `@testing-library/dom`
- [x] B.5 PNGs e assets sem uso (decisão do dono): `public/logo-full.png`, `logo-text.png`, `apple-touch-icon.png`,
      `favicon-16x16.png`, `favicon-32x32.png`, `mstile-150x150.png`, `src/assets/sound.ogg`, e as fontes
      `logo-completa.png` / `watink-logo-letras.png` com suas linhas no `copy-assets.js`. **Ficam** `fundo.png`
      (alimenta `login-background.png`), `watink-sf.png` (alimenta `logo.png`), `favicon.png`. Também saiu
      `public/index.html` (resto do CRA com `%PUBLIC_URL%`, nunca usado: o Vite usa o `index.html` da raiz)
- [x] B.6 `theme/index.ts` + `typography.ts` (saíram com o grupo). Exports mortos isolados **ficam**: o `knip` aponta
      75, mas quase todos são shadcn/`ui/*` e tokens (ruído); remover sem prova individual arrisca quebrar
- [x] B.7 `tsc`, `eslint`, `vitest` e `vite build` (comparar o tamanho do build); PR, CI verde, merge

## PR C: business (`chore/purge-business-dead-code`)

- [x] C.1 `internal/models/user_queue.go` (struct sem uso; a tabela `user_queues` é criada pela tag `many2many` e
      **continua**) e símbolos de ocorrência única: `checkoutRequest`, `ChannelAdapter`, `FlowRunSubjectNone`, `GroupCampaignRunStatusFailed`; `otelTraceParent`/`otelTraceState` também saíram (propagação nunca ligada). `ErrNoRecordingFile` já não existia.
- [x] C.2 `go.mod`: `go mod tidy` não mudou nada (o `go-mp3` já está certo nesta base); corrigir o comentário obsoleto de
      `domain/broadcaster.go:4`
- [x] C.3 **`fetch_url_crawl.go` FICA** (decisão do dono): documentar em `docs/agents/knowledge-base.md` (o que faz,
      limites, estado "implementado, não ligado", como ligar) e registrar no `ORCHESTRATOR-ROADMAP.md` como épico
      "Crawl de site na Base de Conhecimento"
- [x] C.4 **Ficam** (documentado): helpers que só testes usam (`calls`, `pluginlicense`, `plugins.NewPluginManager`),
      `internal/testutil` e o vendor `recording/shine`
- [x] C.5 `go build`, `go vet`, testes dos pacotes tocados; PR, CI verde, merge

## Engine (adiado: follow-up no roadmap)

Nada foi removido do engine nesta change, de propósito. As ~30 funções de `voip/` e `calls/` que só testes usam são
portagem de feature em andamento (`add-whatsapp-video-calls`, ~21 tarefas abertas). A reavaliação ficou registrada
como **Epic 9** em `ORCHESTRATOR-ROADMAP.md`, com a condição para retomar.

## Fechamento

- [x] F.1 Publicar `:test` e validar na stack do dono (a imagem encolhe: `business/web` e os assets saem do repositório); validado pelo dono
- [x] F.2 `/qa-analyst` sobre a DAG concluída: spec x implementação x testes, sem achados bloqueantes (ver `design.md`, "QA")
- [x] F.3 `openspec validate purge-dead-code --strict`
- [x] F.4 Remover os recursos de teste da sessão (`wb-test-pg`, `wb-test-mq`, `wb-test-redis`, banco `repro`)

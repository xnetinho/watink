# Proposal

## Why

O repositório acumulou código, configuração e documentação que descrevem coisas que não existem mais. Isso custa
tempo de leitura, contexto de agente (o `CLAUDE.md` tem mais de 600 linhas e lista serviços removidos) e esconde o
que é real: um bundle compilado versionado (15,8 MB), scripts e workflows apontando para pastas inexistentes,
`dependabot` sem o `engine-go` (nunca recebe atualização), e ~25 componentes de frontend sem nenhum importador.

Esta change **não muda comportamento**. Os bugs de log (`fix-gorm-log-noise`) e de RLS (`fix-auth-rls-set-local`)
são changes separadas e vão antes.

## What Changes

Em lotes pequenos, cada um verificado por build, testes e CI:

- **Repositório:** `business/web/` (bundle estale), `frontend/frontend/` (3 arquivos vazios), `update.sh`,
  `business/run_migrate.go`, `plugins/watink-smtp-go/` (esqueleto de 42 linhas, decisão do dono), 4 scripts sem referência, scripts `windows:*` e `ecosystem.config.js`, workflow que
  dispara em `master`, e `dependabot`/`codeql`/`.gitignore` apontando para pastas inexistentes (e adicionar o
  `engine-go` ao dependabot).
- **Documentação:** `CLAUDE.md` (serviços removidos, lista de plugins), ADR 0018 (superado), um link quebrado,
  `docs/legacy-*` marcados como histórico, `ESTADO_ORQUESTRATOR.md` arquivado.
- **Frontend:** a cadeia antiga do chat, páginas e modais sem rota, ~15 componentes soltos, 4 dependências e os
  PNGs sem uso (decisão do dono).
- **Business:** `UserQueue` e símbolos de uma ocorrência, comentário obsoleto. O `fetch_url_crawl.go` **fica**
  (decisão do dono): é a base para indexar um site inteiro na Base de Conhecimento; será documentado e entra no
  roadmap.

### Fora do escopo

- Código do engine de voz e vídeo: só testes o usam hoje, mas é feature em andamento (`add-whatsapp-video-calls`).
  Reavaliar quando ela fechar.
- Helpers que só testes usam (`calls`, `pluginlicense`, `plugins.NewPluginManager`), `internal/testutil` e o vendor
  `recording/shine`: ficam.
- `publish-ghcr-fork.yml` (publica as imagens `:test` do fork).
- Qualquer mudança de comportamento.

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| Remover componente "órfão" carregado por string ou `lazy` | Baixo | Grafo de imports resolvido por script, mais `tsc`, `vitest` e `vite build` antes de cada lote |
| Remover dependência usada fora de `src/` | Médio | Conferida à mão: `express` é usado por `frontend/server.js` e os plugins do ESLint são carregados por string; ficam |
| Apagar `business/web/` e algo ainda ler dali | Muito baixo | Nenhuma referência em compose, workflow ou Dockerfile; o embed vem de `internal/web/build` |
| Workflows e `dependabot` só se validam na CI | Médio | Abrir PR e ver verde antes do grupo seguinte |

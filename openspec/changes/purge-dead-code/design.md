# Design

## Critérios

- **Remove-se o que um grafo de imports/`deadcode`/`knip` mostra sem uso E uma busca manual confirma.** Ferramenta
  sozinha tem falso positivo (`express` é usado por `frontend/server.js`; os plugins do ESLint são carregados por
  string no `.eslintrc.js`).
- **Código só usado por teste** (helpers de `calls`, `pluginlicense`, `plugins.NewPluginManager`) **fica**: remover
  quebra teste sem ganho de produção.
- **Vendor** (`recording/shine`) e **código de feature em andamento** (engine de voz/vídeo) ficam.
- Cada lote é um commit pequeno com: build, `go vet`, testes do pacote tocado, `tsc`, `eslint`, `vitest` e, no
  frontend, `vite build`. Nenhum lote mistura limpeza com mudança de comportamento.
- Lote de workflows/`dependabot` só é validado de verdade na CI; abrir PR e ver verde antes do próximo.

## Ordem

1. Repositório e documentação (sem risco de runtime).
2. Frontend, por lotes.
3. Business, por lotes.
4. Reavaliar o engine quando a change de vídeo fechar.

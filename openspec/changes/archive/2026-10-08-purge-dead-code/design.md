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

## QA (fechamento)

Confronto da spec com o estado de `origin/develop` e da imagem `:test`, com evidência independente:

| Requisito | Verificação | Resultado |
|---|---|---|
| Remoção segura de código morto | Os 22 itens que deviam sair não existem mais; os que deviam ficar (crawler, `server.js`, `login-background`/`logo`/`favicon`, vendor `shine`, `testutil`, scripts manuais) existem; `express`/`express-rate-limit`/`@testing-library/dom` seguem em `package.json` | ok |
| Configuração aponta só para o que existe | `dependabot` cobre `frontend`, `business`, `engine-go` e a raiz (todo manifesto tem entrada); `codeql`/`.gitignore` sem `legacy`/`marketplace-hub`; `package.json` sem `windows:*` e com scripts existentes | ok |
| Documentação sem serviços removidos | Tabela de serviços do `CLAUDE.md` só tem `business/`, `engine-go/`, `frontend/` (+ plugin-manager em repo privado) | ok |
| Não quebra o build | `go build`/`vet` (business e engine), testes do engine, `tsc`/`eslint`/`lint:ds`, `vitest` 538, suíte completa do business (27 pacotes ok), imagem `:test` sobe, serve o frontend e os 6 PNGs removidos deixam de ser servidos | ok |
| Sem mudança de comportamento | O diff de código de produção do PR C tem 3 linhas adicionadas (todas comentários) e 33 removidas | ok |

Achado fora de escopo, registrado e tratado à parte: `TestServeAudio_VideoNeverReachesTheAudioRecorder` era instável
(~2%), por pedir mais quadros que a fila do bridge retém. Corrigido no PR #7 (só testes).

Nota de verificação: a branch `develop` ainda não tem o código de chamadas (vive em `test/ghcr-images`), então o
conjunto de remoções difere entre as duas (525 em `develop`, 219 em `test/ghcr-images`) sem divergência real: as
diferenças são arquivos que só existem em uma delas.

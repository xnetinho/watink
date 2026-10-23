# Proposal

## Why

Abrir o F12 no navegador mostra a pasta `src/` com **todo o código-fonte TypeScript do frontend**. Não é falha de
compilação: o código é compilado e minificado, mas o build publica junto os **mapas de fonte** (`.map`), e o
navegador usa o mapa para reconstruir o original.

Causa (`frontend/vite.config.mjs:77`): `build.sourcemap: true`, fixo. Medido na imagem `:test` publicada:

- 151 arquivos `.map` embutidos no binário do `business`, 154 no build local (18 MB, contra 6,7 MB sem eles);
- o `business` serve tudo de `/assets/` sem filtro: `GET /assets/index-*.js.map` devolveu **HTTP 200, 3,8 MB**;
- só nesse mapa, **94 arquivos de código próprio** com o conteúdo completo (`sourcesContent`).

Impacto: qualquer visitante, sem login, lê a lógica do frontend, as rotas, os nomes de permissão e os comentários
internos. Não vaza segredo (o código não os contém), mas facilita engenharia reversa e infla a imagem em ~11 MB.

## What Changes

- `build.sourcemap` passa a vir de `buildSourcemap(process.env)` (`frontend/build-sourcemap.mjs`): **desligado por
  padrão**, ligado só com `VITE_SOURCEMAP=true`, num build pontual de depuração. Qualquer outro valor, inclusive um
  typo (`ture`), continua desligado.
- Testes: unidade da função e **build real** (num diretório temporário) provando que o padrão não publica nenhum
  `.map` nem referência `sourceMappingURL`, e que `VITE_SOURCEMAP=true` ainda os gera.

### Fora do escopo

- Filtrar `.map` no servidor (`business/cmd/server/main.go`). É defesa em profundidade, mas o `main.go` não tem testes
  nem ponto de injeção, e extrair isso seria refatoração. Fica registrado no design como follow-up opcional.
- Ofuscar o JavaScript ou remover comentários: o minificador já os remove.
- Guardar os mapas fora da imagem para desminificar erros de produção (sem Sentry hoje; ver design).

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| Depurar um erro de produção fica mais difícil (stack minificada) | Baixo | `VITE_SOURCEMAP=true` num build pontual; hoje não há Sentry nem upload de mapas |
| Algum fluxo dependia dos `.map` | Nenhum encontrado | `git grep sourcemap`: só o `vite.config.mjs`; nenhum workflow, Dockerfile ou código usa |

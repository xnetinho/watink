# Tasks

> Fonte: `proposal.md` e `design.md`. Branch `fix/frontend-sourcemaps`, PR contra `develop`. Todo teste novo é
> provado por mutação.

## 1. Build sem mapas por padrão

- [x] 1.1 Teste que falha hoje: `buildSourcemap` (padrão desligado; `true` liga; qualquer outro valor desligado)
- [x] 1.2 `frontend/build-sourcemap.mjs` e `vite.config.mjs` usando `buildSourcemap(process.env)`
- [x] 1.3 Teste de **build real** (`build-sourcemap.build.test.mjs`, fora de `src/` por usar módulos do Node): por padrão
      nenhum `.map` e nenhum `sourceMappingURL`; com `VITE_SOURCEMAP=true` os mapas existem
- [x] 1.4 Mutação: padrão ligado, aceitar qualquer valor, não aparar espaços e o `vite.config` voltar ao `true` fixo
      derrubam um teste cada

## 2. Documentação e CI

- [x] 2.1 Documentar `VITE_SOURCEMAP` em `docs/dev/commands.md`
- [x] 2.2 CI (`build-frontend`): passo que roda os dois testes; a CI não rodava `vitest`, então sem isso nada pegaria uma regressão

## 3. Verificação e entrega

- [x] 3.1 `tsc`, `eslint` e `vitest` completos; `vite build` (6,7 MB contra 18,1 MB antes)
- [ ] 3.2 Ao vivo na imagem publicada: o `.map` deixa de existir e o F12 não mostra `src/`
- [x] 3.3 `openspec validate fix-frontend-sourcemaps --strict`; commit; PR contra `develop`

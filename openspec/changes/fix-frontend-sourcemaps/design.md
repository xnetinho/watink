# Design

## Causa

`vite.config.mjs` fixava `sourcemap: true`. O `Dockerfile.business` roda `npm run build`, copia `frontend/build`
para `internal/web/build` e o `go:embed` leva tudo para o binário; `main.go` abre qualquer arquivo existente no FS
embutido. Logo, cada `.map` do build é um arquivo público.

## Decisão (aprovada pelo dono)

Desligar por padrão, com chave explícita para depuração.

```js
export function buildSourcemap(env) {
  return String(env.VITE_SOURCEMAP ?? "").trim().toLowerCase() === "true";
}
```

- **Só `true` liga.** Um typo em variável de ambiente nunca pode publicar o código: `ture`, `1`, `yes`, `on` ficam
  desligados.
- **Arquivo próprio** (`build-sourcemap.mjs`) para o `vitest` importar sem executar o `vite.config` inteiro.

## Alternativas descartadas

- **Gerar e remover na imagem** (`sourcemap: 'hidden'` + apagar no Dockerfile, ou guardar como artefato da CI):
  permite desminificar stack traces depois, mas exige guardar os mapas, casar cada build com seu mapa e uma etapa a
  mais. Sem Sentry hoje, não paga o custo. Reavaliar se um erro de produção for difícil de ler.
- **`sourcemap: 'hidden'`**: gera os `.map` mas não escreve a referência nos `.js`. **Não resolve**: os arquivos
  continuam na pasta publicada e acessíveis por URL.
- **Filtro no servidor**: ver abaixo.

## Defesa em profundidade (follow-up opcional, fora desta change)

`business/cmd/server/main.go` poderia responder 404 a `*.map`, para o caso de alguém reativar o mapa e esquecer. Não
foi feito porque a função está dentro de `main()`, sem teste e sem ponto de injeção. Extrair para um handler testável
é uma refatoração pequena, mas é escopo diferente de "o build publica mapas".

## Verificação

- Unidade: `buildSourcemap` (padrão, `true` em qualquer caixa e com espaços, valores inválidos).
- **Build real**: roda `vite build` num diretório temporário e confere o que seria publicado: zero `.map` e zero
  `sourceMappingURL` por padrão; com `VITE_SOURCEMAP=true` os mapas existem. Pega o caso que a unidade não pega: o
  `vite.config.mjs` voltar a `sourcemap: true` sem usar a função (mutação 4).
- Ao vivo: imagem publicada, `GET /assets/index-*.js.map` deve devolver o `index.html` (rota desconhecida cai no
  fallback da SPA) e não o mapa; o F12 não mostra a pasta `src/`.

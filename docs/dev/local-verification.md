# Verificação local (portão 1) e túnel para validação

Como uma mudança é verificada **antes** de subir para o GitHub. Complementa o pipeline de
[`git_workflow_policy.md`](git_workflow_policy.md) (local → homologação → produção).

> **Princípio:** só sobe para o GitHub o que já foi construído, executado e verificado localmente. A CI do
> GitHub continua sendo o árbitro final antes do merge, não o primeiro lugar onde o código é compilado.

## Por que isto existe (e o que NÃO é o motivo)

- **Não é custo.** O repositório é público e os minutos de Actions são gratuitos; um build de imagem leva ~2,5 min.
- **É o ciclo de feedback e o gate.** O fluxo anterior (branch longa `test/ghcr-images` que publicava imagens a cada
  push) deixou **69 commits sem passar pela CI** por 3 semanas; ao chegar no `develop`, o linter acusou 58 achados de
  uma vez. Verificar localmente antes de cada push evita isso.

## O fluxo

```
1. implementar na branch (feat/ fix/ ...)
2. LINT + TESTES locais ............ obrigatório antes de qualquer push
3. BUILD das imagens + stack local . constrói as imagens DESTE checkout e exercita o fluxo
4. (opcional) túnel Cloudflare ..... validação humana contra o stack local, no que o agente não alcança
5. push + PR → develop ............. a CI do GitHub confirma; só então merge
```

## 1. Lint e testes (sempre)

```bash
# Go: o MESMO linter da CI (golangci-lint v2.12.2, config em business/.golangci.yml)
cd business && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run ./...
cd business && go build ./... && go vet ./... && go test ./... -short -count=1
cd engine-go && go vet ./... && go test ./... -race -count=1

# Frontend
cd frontend && npm run typecheck && npm run lint && npm run lint:ds && npx vitest run
```

- O linter v2.12.2 exige Go 1.25. Em máquina com Go mais antigo, `GOTOOLCHAIN=auto` baixa o toolchain certo
  sozinho (foi o que destravou rodar o linter da CI localmente). Sem isso, `go run ...golangci-lint@v1` e
  `staticcheck` antigos falham ao carregar o `.golangci.yml`.
- A CI usa `only-new-issues: true`. Em PRs com **mais de 300 arquivos** a API do GitHub recusa o diff e o linter varre
  o módulo inteiro: rode o lint completo local para não ser surpreendido. Prefira PRs pequenos.
- Os testes do business precisam de **Postgres com `pgvector`** e RabbitMQ (`DB_HOST`, `DB_PORT`, `DB_PASS`,
  `AMQP_TEST_URL`); sem eles os testes de repositório falham por conexão, não por defeito.

## 2. Stack local: construir e subir

`docker-compose.local.yml` constrói as imagens **deste checkout** (Postgres, business, engine) e sobe o stack mínimo
em rede e volumes próprios (projeto `lt`), isolado de homologação e produção.

```bash
cp .env.local.example .env.local     # segredos DESCARTÁVEIS; o .env.local é ignorado pelo git
docker compose -p lt -f docker-compose.yml -f docker-compose.local.yml --env-file .env.local up -d --build

curl -s http://localhost:8082/api/health        # {"service":"watink-business","status":"OK"}
curl -s http://localhost:8082/api/v1/initial-setup/check   # {"needsSetup":true} em banco novo

docker compose -p lt -f docker-compose.yml -f docker-compose.local.yml --env-file .env.local down -v   # descarta tudo
```

Medido em máquina de 4 CPUs e 4 GB: build do business 2m27s, do engine 1m28s; o stack completo em repouso cabe em
~1 GB e deixa folga. **Não rode o stack junto de um navegador de teste e de uma suíte completa ao mesmo tempo**
em máquina de 4 GB.

### O que dá para verificar no stack local

Tudo que **não depende do WhatsApp real**: migração do banco (94 tabelas), wizard e login, rotas protegidas,
log do servidor (sem `syntax error`, sem `record not found`, sem SQL com valores), SSE, filas RabbitMQ, a UI no
navegador (F12 sem mapas de fonte, citação de resposta, ack), regressões de fluxo.

### O que NÃO dá para verificar só com o stack local

- **Chamadas de voz e vídeo.** O áudio sai por **UDP direto do host do engine** para os relays do WhatsApp, com o IP
  público de quem hospeda o engine. Pareando o número de teste neste stack você prova que a chamada **funciona**; não
  prova a **qualidade nem o comportamento no ambiente de produção** (NAT, firewall, UDP de saída). Isso continua sendo
  validado na homologação.
- **Efeito sobre o risco de ban** do número (ADR 0016): depende de tempo de uso, não de um teste.

## 3. Túnel Cloudflare para validação humana

Para validar contra o stack local pelo navegador de quem não está na máquina:

1. O business escuta em `0.0.0.0:8082`. Aponte o `cloudflared` (numa VM da mesma rede) para
   `http://<ip-da-maquina-do-stack>:8082`.
2. O túnel deve **desligar buffering** na rota SSE `/api/v1/events` (o business já envia `X-Accel-Buffering: no`) e
   aceitar **WebSocket** em `/api/v1/calls/:id/audio`.
3. O canal de áudio do engine (porta **8085**) é interno e **sem autenticação própria**: **nunca** o exponha no túnel
   nem em `ports:`.

### Número de WhatsApp de teste

- Use um **número exclusivo de teste**, pareado **somente** neste stack. A sessão do WhatsApp fica no **Postgres**
  (não em volume do engine), então parear aqui **não afeta** a sessão de produção, **desde que o número seja outro**.
- **Nunca** pareie o mesmo número em dois lugares: os dois se desconectam mutuamente.
- A sessão vive em `lt_postgres_data`. Se esse volume for removido (`down -v`), é preciso parear de novo por QR.

## 4. Quando publicar uma imagem

Imagens `:test` no GHCR deixam de ser publicadas a cada push. O workflow do GHCR continua com `workflow_dispatch`:
dispare-o **manualmente**, a partir da branch de feature, só quando for preciso que a imagem rode **fora** desta
máquina (por exemplo, para validar chamadas no IP e na rede reais).

## Checklist antes de abrir o PR

- [ ] Lint completo (mesma versão da CI) com 0 achados
- [ ] Testes do que foi tocado, e a suíte completa se o diff for grande
- [ ] Imagens construídas e stack local no ar; fluxo exercitado
- [ ] Log do servidor limpo
- [ ] Evidência (saída do lint/teste/curl) colada na descrição do PR

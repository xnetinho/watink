# Git Workflow Policy (Watink)

## Regras obrigatórias

1. **Proibido commit direto em `main`** — todo trabalho via PR.
2. Toda branch deve virar PR com testes antes de merge.
3. Merge só após CI/smoke e revisão humana.
4. Commits seguem **Conventional Commits**.

## Convenção de Branches

Baseada no tipo de mudança (espelha o prefixo do commit):

| Prefixo | Uso | Exemplo |
|---|---|---|
| `feat/<tema>` | Nova funcionalidade | `feat/helpdesk-kanban` |
| `fix/<tema>` | Correção de bug | `fix/ticket-status-update` |
| `refactor/<tema>` | Refatoração sem nova feature | `refactor/frontend-shadcn-migration` |
| `chore/<tema>` | Manutenção, tooling, deps | `chore/update-go-deps` |
| `docs/<tema>` | Somente documentação | `docs/adr-di-backend` |
| `hotfix/<tema>` | Correção urgente em produção | `hotfix/rabbitmq-reconnect` |

## Conventional Commits

```
feat:      nova funcionalidade
fix:       correção de bug
refactor:  mudança de código sem alterar comportamento
chore:     manutenção, tooling, deps
docs:      somente documentação
hardening: segurança, resiliência
test:      adição ou correção de testes
```

## Fluxo Padrão

```bash
git fetch origin && git checkout main && git pull
git checkout -b feat/<tema>
# implementar + commits pequenos
git push origin feat/<tema>
# abrir PR → develop (ou main se develop indisponível)
```

## Checklist de PR (obrigatório)

- [ ] Resumo técnico do que mudou
- [ ] Risco/impacto
- [ ] Evidência de teste (logs/smoke/build)
- [ ] Plano de rollback

## Merge Flow

```
feat/* / fix/* / refactor/* → develop → main (release)
hotfix/*                    → main → back-merge para develop
```

## Pipeline de Ambientes (local → homologação → produção)

O merge flow de branches acima roda dentro de um pipeline de **três estágios com dois portões de validação** — nada não-validado avança:

```
1. DEV LOCAL         implementa na branch por convenção (feat/ fix/ ...)
      │
      ▼
2. VALIDAÇÃO LOCAL   ⟵ PORTÃO 1   (detalhes em local-verification.md)
      │              lint completo com o MESMO linter da CI (golangci-lint v2.12.2)
      │              go build ./... && go test ./...   (business, engine-go)
      │              npm run build / typecheck / lint  (frontend)
      │              construir as imagens e subir o stack local; exercitar o fluxo
      │              (opcional) túnel Cloudflare para validação humana
      ▼
3. HOMOLOGAÇÃO       PR → develop → deploy AUTOMÁTICO no ambiente de homologação
      │              ambiente: homolog.watink.com
      │              validar/aprovar o comportamento em ambiente real
      │              ⟵ PORTÃO 2
      ▼
4. PRODUÇÃO          develop → main → deploy AUTOMÁTICO em produção
```

**Regras:**

- **Homologação rastreia `develop`**; **produção rastreia `main`**. Cada promoção exige o portão anterior verde.
- **Nunca** promover para homologação sem validação local, nem para produção sem aprovação em homologação.
- **Deploy é automático nos dois estágios** via GitHub Actions
  (`.github/workflows/cd-homolog.yaml` e `cd-production.yaml`): um push em
  `develop`/`main` builda a imagem e faz o deploy via SSH direto na VPS
  (`git reset --hard` no branch + `docker compose build && up -d`), seguido
  de smoke test (`GET /api/health`). Nenhum passo manual depois do merge.
- **Produção roda em `app.watink.com`**, na mesma VPS de homolog, como um
  stack Docker isolado (containers, rede, volumes e segredos próprios —
  nunca compartilhados com homolog). Ver `docker-compose.prod.yml` na raiz
  do repo — o mesmo arquivo serve os dois ambientes, diferenciados só pelo
  `.env`/`.env.prod` e pelo project name do Compose (`-p watink-homolog` vs
  `-p watink-prod`).
- **Não há branch longa de teste.** Não se mantém uma branch paralela (como foi `test/ghcr-images`) que publica imagens a
  cada push: ela acumulou 69 commits sem passar pela CI por 3 semanas. Cada mudança vive numa branch de feature
  curta, é verificada localmente (portão 1), vira PR contra `develop` e só então ganha imagem. Publicar uma imagem
  `:test` é **sob demanda** (`workflow_dispatch`), ver `local-verification.md`.
- Reportar honestamente o resultado de cada portão (build/testes/homologação) — não marcar "aprovado" sem evidência.

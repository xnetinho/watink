# ADR 0034 — Isolamento entre empresas imposto pelo banco (RLS real, papéis e escopo)

**Status:** Accepted (substitui o ADR 0001 no que diz respeito ao mecanismo de isolamento)
**Data:** 2026-10-23
**Change:** `openspec/changes/enforce-tenant-rls`

## Contexto

O ADR 0001 admitia que o RLS nunca funcionou: o app conectava como `postgres` (superusuário, `BYPASSRLS`), só 10 das 59
tabelas com `tenantId` tinham política e nada definia o tenant na sessão. O isolamento era um `WHERE "tenantId"` escrito à
mão em cada consulta (239 pontos de `GetScoped` mais SQL manual em workers). Um filtro esquecido vazava dados de outra
empresa sem nenhum erro.

## Decisão

1. **Três papéis, dono separado.** `watink_app` (`NOSUPERUSER NOBYPASSRLS`, só DML) atende toda requisição e todo evento
   com empresa; `watink_system` (`NOSUPERUSER BYPASSRLS`, só DML) é um segundo pool para caminhos sem empresa; o
   administrador (dono dos objetos) roda **apenas** `watink-business migrate`. O servidor recusa subir com credencial de
   superusuário ou `BYPASSRLS` e exige `DB_APP_PASS` e `DB_SYSTEM_PASS`.
2. **Escopo por contexto.** `pkg/tenantdb.Router` é a `gorm.ConnPool` do handle global. Cada comando é roteado pelo escopo
   do `ctx`: empresa (conexão fixada com `app.tenant_id` definido), sistema (pool `watink_system`) ou nenhum. Sem escopo, uma
   consulta a tabela com dados de empresa falha com `ErrNoScope` (ruidoso), em vez de devolver zero linhas em silêncio.
3. **Política gerada do catálogo.** `tenantdb.ApplyPolicies` cria, em toda tabela com `tenantId`, `ENABLE ROW LEVEL SECURITY` e
   uma política única `USING`/`WITH CHECK` sobre `NULLIF(current_setting('app.tenant_id', true), '')::uuid` (sem empresa, a
   comparação é com `NULL` e nunca é verdadeira: falha fechada). Tabelas filhas sem `tenantId` herdam o isolamento do pai por
   `EXISTS` (`tenantdb.Children`); `Tenants` usa a própria chave; as realmente globais ficam em `tenantdb.Exempt`.
4. **Pontos de entrada abrem o escopo; o resto propaga o `ctx`.** `IsAuth` (empresa do token), eventos AMQP (empresa do
   envelope), webhook (descobre a conexão no escopo de sistema e processa no da empresa), crons (tick de sistema, trabalho
   por empresa), rotas públicas e internas (sistema). `Release()` devolve a conexão antes de chamada externa lenta; `Detach`
   dá um escopo independente a goroutines que sobrevivem à requisição; `safego` contém pânico.
5. **Guardrails executáveis:** `TestCatalogo_TodaTabelaComTenantIdTemPoliticaAtiva` (tabela nova sem política quebra o CI),
   `TestVazamento_PorTabelaGeradoDoCatalogo` (A não lê nem escreve linha da B em cada tabela) e
   `TestEscopoDeSistema_SoPacotesAutorizados` (quem pode abrir o escopo de sistema é uma lista fechada e justificada).

### Divergência do desenho original (D3)

O `design.md` previa transação preguiçosa por requisição com `set_config(..., true)`. Foi trocada por **conexão fixada por
escopo com `set_config(..., false)` e `RESET` na devolução** (se o `RESET` falhar, a conexão é descartada). Motivo: a
transação por requisição exigiria commit/rollback decidido pelo status HTTP e abortaria o restante da requisição a cada
violação de unicidade tratada como fluxo normal (14 pontos), sem ganho de isolamento. A semântica transacional existente
(`Transaction`/`SAVEPOINT` do GORM) fica intacta.

## Evidência medida

Banco com 40 mil contatos e 40 mil tickets (20 mil por empresa), 300 repetições por consulta, mesmos dados, mesmo filtro
explícito, RLS ligado (papel app) contra desligado (superusuário):

| Consulta | RLS ligado | RLS desligado |
|---|---|---|
| tickets abertos (lista, 50) | 4,105 ms | 4,044 ms |
| contatos por busca de nome | 16,2 ms | 26,0 ms |
| contagem de tickets por status | 3,390 ms | 2,919 ms |
| ticket por id | 0,222 ms | 0,156 ms |
| estágios (tabela filha via `EXISTS`) | 0,285 ms | 0,081 ms |

A busca por nome ficou **mais rápida** com RLS (reproduzido em 3 rodadas alternadas): o planejador avalia o predicado de
empresa da política antes do `ILIKE` caro (`One-Time Filter` no plano, avaliado uma vez), enquanto sem RLS avalia o `ILIKE`
em cada linha antes do filtro de empresa. Não é cache: é a ordem dos predicados. O custo nas consultas indexadas é
desprezível (≈ 0,05 a 0,5 ms). A tabela filha paga mais em termos relativos porque o
`EXISTS` faz varredura no pai; com o volume real do pai (dezenas de pipelines por empresa) isso fica em microssegundos, mas
é o ponto a observar se alguma filha crescer muito (alternativa: coluna `tenantId` na filha).

## Consequências

- **BREAKING:** variáveis de ambiente novas (`DB_APP_PASS`, `DB_SYSTEM_PASS`, `DB_ADMIN_USER`/`DB_ADMIN_PASS` só no job de
  migração). O esquema deixa de ser migrado no boot do servidor: roda `watink-business migrate` antes.
- Os filtros `"tenantId"` existentes continuam como defesa em profundidade e por desempenho (índice); deixam de ser a única
  barreira.
- Todo código novo que acesse o banco fora de uma requisição precisa de `tenantdb.WithTenant`/`TenantCtx` ou, justificadamente,
  do escopo de sistema (lista fechada).
- Um caminho esquecido falha com `ErrNoScope` (e log `SEM ESCOPO em arquivo:linha`), nunca vaza.

## Limites conhecidos

- O engine (`engine-go`) ainda conecta como `postgres`; a troca para `watink_engine` está na tarefa 6.2 do change.
- `ErrNoScope` detecta tabelas por token no texto do SQL; um nome de tabela de empresa em string literal poderia gerar falso
  positivo (falha ruidosa, nunca vazamento).
- Mensagens (`Messages`) e outras tabelas com colunas obrigatórias sem default não entram no teste de vazamento genérico
  (11 de 59); são cobertas pelos testes de isolamento de repositório e pela política idêntica gerada do catálogo.

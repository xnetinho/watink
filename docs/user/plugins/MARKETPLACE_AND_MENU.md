# Marketplace e menu lateral

Este guia explica como o **Marketplace** de plugins e o **menu lateral** se relacionam: onde
cada coisa fica, o que o botão **Ativar** faz de verdade e por que às vezes um recurso novo
não aparece no menu.

## Onde fica

| O quê | Onde |
|---|---|
| Marketplace | **Configurações → Marketplace de Plugins** (botão no topo; exige a permissão `settings:update`). Rota: `/admin/settings/marketplace` |
| Detalhe de um plugin | `/admin/settings/marketplace/:slug` |
| Menu lateral | Barra à esquerda de toda tela (componente `SidebarNav`) |

## Plugin `free` × `pro`

- **`free`**: gratuito. Ativar não consulta licença nenhuma; basta clicar.
- **`pro`**: pago. Para ativar, a **instância** precisa ter uma licença válida e haver vaga
  no teto de empresas (tenants) da licença. Sem licença, **Ativar** responde que o plugin
  precisa de checkout (erro `plugin_unlicensed`) e você conclui a compra no próprio
  Marketplace antes de tentar de novo.

Quem decide se um plugin é `free` ou `pro` é o **catálogo do Hub**, não o código do sistema.

## As duas abas

- **Meus Plugins**: os que a sua empresa já ativou.
- **Catálogo**: tudo o que ainda pode ser ativado ou comprado (com carrinho para os pagos).

Se o catálogo vier vazio com o aviso de **offline**, o sistema não conseguiu falar com o Hub
naquele momento; nada foi perdido, tente de novo em alguns instantes.

## O que o botão Ativar faz

1. Confere se o plugin é `free` (catálogo) ou `pro` (licença).
2. Em instâncias geridas pelo Watink SaaS, confere também se o **plano** da sua conta inclui o plugin.
3. Se for `pro`, confere a licença e o teto de empresas.
4. Grava a **alocação** do plugin para a sua empresa (`PluginInstallations`) e responde `{ "active": true }`.

A alocação diz **qual empresa usa o plugin**. Ela **não** é a prova de licença: a licença é
verificada no servidor a cada uso. **Desativar** marca a alocação como inativa e preserva o
histórico.

## Como o menu lateral decide o que mostrar

O menu pergunta ao servidor quais plugins a sua empresa tem ativos (`GET /plugins/installed`,
campo `active`) e repete a pergunta a cada **5 minutos**. Os itens que dependem de plugin
aparecem quando o `slug` correspondente está nessa lista:

| Item do menu | Rota | `slug` exigido | Permissão |
|---|---|---|---|
| Helpdesk | `/helpdesk` | `helpdesk` | `tickets:read` |
| Agentes de IA (Assistentes) | `/assistants` | `assistant` | `pipelines:read` |
| Grupos WhatsApp | `/grupos-whatsapp` | `groups` | `whatsappGroups:read` |

O item só aparece quando o plugin está ativo **e** o usuário tem a permissão da coluna.

Recursos do **núcleo** (tickets, contatos, pipelines, atividades, **chamadas**, etc.) **não**
dependem de plugin: o menu os mostra só pela **permissão** do usuário.

## Por que um plugin `free` novo não aparece no Marketplace

O Marketplace lista **o catálogo do Hub**. Um plugin que existe só no código do sistema, sem
entrada no catálogo, **não é listado**, e portanto não pode ser ativado por essa tela. Para um
recurso `free` aparecer:

1. cadastre o plugin no catálogo do Hub (com o mesmo `slug` do código), com o tipo `free`;
2. publique-o (um plugin em rascunho não aparece);
3. recarregue o Marketplace.

Isso vale para plugins embarcados como o de **Grupos e Comunidades**, que fica em rascunho no
catálogo até o preço ser definido.

> Por isso as **Chamadas de voz** não são um plugin: não precisam de entrada no Hub nem de
> clique em Ativar. O acesso é por permissão (ver [Habilitar chamadas](../calls/ENABLING_CALLS.md)).

## Onde isso está no código

- Página: `frontend/src/pages/Marketplace/` (abas em `index.tsx`, dados em `hooks/useMarketplace.ts`)
- Menu: `frontend/src/components/MainSidebar/components/SidebarNav.tsx` e `hooks/useMainSidebar.ts`
- Rotas do servidor: `business/internal/routes/routes.go` (`/plugins/catalog`, `/plugins/installed`,
  `/plugins/:slug/activate`, `/plugins/:slug/deactivate`, `/plugins/instance`)
- Regra de ativação: `PluginController.Activate` em `business/internal/controllers/plugin_manager.go`

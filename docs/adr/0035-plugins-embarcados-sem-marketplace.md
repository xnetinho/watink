# ADR 0035 — Plugins embarcados sem marketplace nem licença

**Status:** Accepted (supera os ADRs 0024, 0025 e 0026)
**Data:** 2026-10-23

## Contexto

Os ADRs 0024 a 0026 desenharam um marketplace de plugins com licenciamento via Hub (token Ed25519 verificado por um
`plugin-manager` proprietário), compra por cartão e Pix, e um modo gerido pelo Watink SaaS. O `plugin-manager` é um
componente de código fechado, distribuído só como imagem de um repositório privado do upstream; o fork não tem a fonte
nem como publicá-lo. O resultado prático no fork era um marketplace que não funcionava: sem o serviço, o gate de licença
falhava de forma fechada e os cinco plugins embarcados (Helpdesk, Webchat, Assistentes de IA, Grupos e Estoque Avançado)
ficavam bloqueados com 402.

## Decisão

1. **O marketplace local deixa de existir.** Saem: catálogo, checkout (cartão, Pix, carrinho), instância, licença, a rota
   interna `/internal/plugin-manager`, o pacote `pluginlicense`, o serviço `plugin-manager` dos compose e o `pull` dele no
   deploy. A tela "Marketplace" é substituída por **Configurações > Plugins**.
2. **O único gate de runtime é a alocação por empresa** (`PluginInstallations.active`). O administrador da empresa liga ou
   desliga cada plugin embarcado (`POST /plugins/:slug/activate|deactivate`); `GET /plugins/installed` lista os ativos.
   Não há mais estado somente-leitura vindo de licença nem teto de empresas por plugin.
3. **Só plugins embarcados são ativáveis.** `plugins.EmbeddedSlugs()` é a lista; um slug desconhecido responde 404 e não
   grava nada.
4. **Fora do escopo:** o modo `plan_only`/`catalog_visible` do ADR 0026 e a reconciliação de downgrade por plano
   (`reconcileDeactivations`) saem junto, por dependerem do catálogo. A coluna `Plans.pluginEntitlements` e a tabela
   `InstancePolicies` permanecem no esquema por serem escritas pelo sincronismo com o Watink SaaS; não são mais lidas
   para decidir acesso a plugin.

## Consequências

- Todos os cinco plugins ficam disponíveis a qualquer empresa que os ative. **Não há mais cobrança nem limite por plano
  para plugins.** Se isso voltar a ser necessário, precisa de um novo desenho (licença local própria), e não da
  reativação do `plugin-manager`.
- A tela e os endpoints de compra deixam de existir (**BREAKING** para quem os consumia); o frontend tem redirecionamento
  de `/admin/settings/marketplace` para `/admin/settings/plugins`.
- O compose de produção perde um serviço e a variável `PLUGIN_MANAGER_INTERNAL_TOKEN`.
- O ADR 0003 (flag de plugin embutido) continua válido como histórico; os ADRs 0024, 0025 e 0026 ficam como registro do
  desenho anterior.

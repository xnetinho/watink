# Proposal

## Why

A OLX é um dos maiores marketplaces de classificados do Brasil (veículos, imóveis, usados). O vendedor recebe contatos
pelo **chat do anúncio** e hoje responde no site/app da OLX, fora do Watink. A OLX oferece uma **API de chat para CRMs
integradores** (OAuth + webhook) que permite receber e responder dentro do sistema do vendedor, ganhando fila, SLA,
funil e histórico unificado.

É o canal que **mais entrega dados do contato** entre os marketplaces: o webhook traz **nome, e-mail e telefone** do
comprador, o que permite vincular a um Cliente do CRM.

## What Changes

- Canal `olx` como adaptador de `domain.Channel` usando a API de **Integração de Chat** da OLX.
- **Conexão por OAuth** (`scope=chat`) do anunciante; token cifrado e renovado.
- **Recepção por webhook**: o sistema registra a URL do tenant via `POST /autoservice/v1/chat` e passa a receber cada
  mensagem do comprador com `chatId`, `messageId`, `listId`, nome, e-mail e telefone.
- **Envio** de texto via `POST /autoservice/v1/chat/send` (`chatId`, `messageId`, `textMessage`).
- **Uma conversa por anúncio e chat** (`conversationKey = listId:chatId`) e contexto do anúncio no ticket.
- Preenchimento do `Contact` (nome, e-mail, telefone) a partir do webhook.

### Fora do escopo
- **Leads** (`Integração de Leads`) e **importação/gestão de anúncios**: são outras APIs da OLX.
- Iniciar conversa (a API só responde a chats que o comprador abriu).
- Mídia: **só texto está documentado** no envio; imagens/anexos ficam fora até haver confirmação.
- Histórico veicular e renovação de anúncios.

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| **Acesso à API não é aberto:** o `client_id` é "fornecido pela OLX durante o registro da aplicação"; não está claro se qualquer SaaS consegue registrar um app ou só integradores homologados | **Alto (bloqueia a change)** | Tarefa 0.1: contatar `suporteintegrador@olxbr.com` **antes** de implementar; sem resposta positiva a change não prossegue |
| O webhook é **por conta do anunciante**: cada tenant registra a própria URL; uma URL por conexão | Médio | URL com `connectionId` e segredo por conexão (a OLX **não documenta assinatura** do webhook) |
| **Sem assinatura documentada** no webhook: só dá para restringir por IP de saída da OLX (54.162.151.93) | **Alto** | Segredo no caminho da URL + restrição por IP configurável + validação do `chatId` contra conexões do tenant; documentar o limite |
| Só **texto** documentado; limite de caracteres e janela **não documentados** | Médio | Tratar como desconhecido: não prometer mídia nem prazo; descobrir na prática com a conta de teste e registrar |
| Rate limit **5.000 req/min por IP** com bloqueio de 10 min (429) | Médio | O limite é **por IP do Watink**, compartilhado entre todos os tenants: token-bucket global |
| Há **inconsistência** na doc (Leads diz que o anunciante ainda responde pelo portal; Chat marca "responder CRM→OLX" como Disponível) | Médio | Validar o envio com conta real antes de prometer |
| Mensagens `senderType` = `system` e `origin` = `seller` (eco das nossas respostas) | Baixo | Ignorar `origin=seller` e `senderType=system` para não duplicar o que o próprio sistema enviou |

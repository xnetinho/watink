# Design

Depende de `add-channel-abstraction`. Fonte: `developers.olx.com.br/chat/*` lida em out/2026.

## Conexão

OAuth (`auth.olx.com.br/oauth`, `scope=chat`); `credentialsEnc = {access_token, refresh_token, expires_at}`. Ao conectar,
o sistema ativa o webhook: `POST https://apps.olx.com.br/autoservice/v1/chat` com `{"webhook": "<url da conexão>"}`
(201 = criado, 200 = atualizado, 401 = token inválido). Ao desconectar: `DELETE` no mesmo endpoint.
**Pendente:** como obter `client_id`/`client_secret` (registro de aplicação) e se há homologação para SaaS.

## Capacidades

```go
Capabilities{ InitiateConversation:false, ReplyWindow:nil /* nao documentada */, MaxTextBytes:0 /* desconhecido */,
  Media:[] /* so texto documentado */, Groups:false, Typing:false, ReadReceipts:false, Auth:oauth, HasPhone:true }
```

`MaxTextBytes=0` significa "desconhecido": o servidor aplica um teto conservador configurável e **registra** o que a OLX
recusar, em vez de inventar um número.

## Payload do webhook (lido da doc)

```json
{ "chatId": "...", "message": "...", "senderType": "buyer|account|system",
  "email": "...", "name": "...", "phone": "...", "messageTimestamp": "2006-01-02T15:04:05.000",
  "messageId": "...", "origin": "buyer|seller", "listId": "..." }
```

| OLX | Watink |
|---|---|
| `messageId` | `externalMessageId` (dedup) |
| `listId` + `chatId` | `conversationKey = listId:chatId` |
| `name`, `email`, `phone` | `Contact` (nome, e-mail) e `ContactIdentity.meta.phone`; **vincular a Cliente do CRM** se o telefone/e-mail bater (manual, ADR 0023) |
| `origin = seller` ou `senderType = system` | ignorado (eco do que o próprio sistema/anunciante enviou) |
| `messageTimestamp` | `createdAt` (formato sem fuso: assumir o fuso de Brasília e **registrar** a suposição) |

## Pontos de atenção

- **Autenticidade do webhook:** a OLX não documenta assinatura. Defesa em camadas: segredo longo no caminho da URL,
  allowlist do IP de saída da OLX (configurável) e conferência de que o `chatId`/`listId` pertence à conexão.
- **Compartilhamento do rate limit por IP:** todos os tenants saem pelo mesmo IP; um único token-bucket global.
- **Envio idempotente:** o `messageId` do envio é gerado por nós (`messageId` no corpo), o que dá idempotência; guardar.
- **Contexto do anúncio:** a doc não oferece um endpoint de detalhe do anúncio no chat; o título/preço vêm de outra API
  (importação de anúncios). Primeira entrega mostra só o `listId` e um link.

## Verificação

Contrato + servidor OLX simulado (OAuth, ativação/desativação do webhook, mensagem do comprador, eco `origin=seller`,
401, 429). **Validação ao vivo depende de aprovação da OLX** (tarefa 0.1): sem ela só a simulação é possível.

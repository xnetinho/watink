# Design

Depende de `add-channel-abstraction`. **Fonte e limite:** leitura anterior da documentação oficial da Meta
(Instagram Platform, Graph API v25.0); **não reconfirmada** nesta rodada (as páginas devolvem 400 a `WebFetch`).
A tarefa 0.1 reconfirma tudo antes de codificar.

## Duas rotas, uma escolhida

| | Instagram Login (**escolhida**) | Messenger Platform for Instagram |
|---|---|---|
| Exige Página do Facebook | **não** | sim |
| Host | `graph.instagram.com` | `graph.facebook.com` |
| Permissão | `instagram_business_manage_messages` | `instagram_manage_messages` |
| Token | do usuário do Instagram (60 dias, renovável) | token de Página |

Escolha: Instagram Login, porque o cliente típico (loja pequena) **não tem Página do Facebook vinculada** e esse é o
maior atrito de onboarding. A rota por Página fica como segundo adaptador **somente se** houver demanda.

## Conexão

1. OAuth "Business Login for Instagram": o usuário autoriza; o `code` expira em **1 h** e é trocado por token de curta
   duração e depois por **longa duração (60 dias)**.
2. `credentialsEnc = {access_token, expires_at, igUserId}`; `config = {username, apiVersion}`.
3. Assinar os webhooks da conta (`messages`, `messaging_postbacks`, `messaging_seen`, `messaging_reactions`).
4. **Refresh** por job antes do vencimento (a renovação exige o token ainda válido); falha → `status=reauth`.

## Webhook

- `GET` de verificação: `hub.mode=subscribe`, `hub.verify_token` (segredo da conexão) e eco de `hub.challenge`.
- `POST`: validar `X-Hub-Signature-256` = `sha256=` + HMAC-SHA256 do corpo **bruto** com o **App Secret** (segredo do app,
  não da conexão). Comparar em tempo constante; recusar sem corpo de diagnóstico.
- Dedup por `mid` (id da mensagem).

## Capacidades

```go
Capabilities{ InitiateConversation:false,
  ReplyWindow: 24h, ReplyWindowExtension: &{Tag:"human_agent", Extends:7*24h, HumanOnly:true},
  MaxTextBytes:1000, Media:[image,audio,video,pdf], Groups:false, Typing:true, ReadReceipts:true,
  Reactions:true, Auth:oauth_review, HasPhone:false }
```

## Regra de resposta

`lastInboundAt` do usuário define a janela. `CanReply` devolve: dentro das 24 h = ok; entre 24 h e 7 d = ok **somente**
com a tag `human_agent` e ação de um atendente humano (nunca automação); acima de 7 d = bloqueado, com o motivo. O
composer mostra o tempo restante e, expirado, oferece "enviar como atendente humano" se couber.

## Mapeamento

| Instagram | Watink |
|---|---|
| `sender.id` (IGSID) | `ContactIdentity.externalId` e `conversationKey` |
| `message.mid` | `externalMessageId` |
| `message.attachments` (image/audio/video/file, `payload.url`) | mídia (baixar para `mediastore`; a URL **expira**) |
| `reply_to.mid` | citação |
| `postback.payload` / `quick_reply.payload` | evento de interação |
| `referral` / `comment_id` (resposta privada) | origem da conversa (anúncio, post) |

## Verificação

Contrato + servidor Meta simulado (desafio, assinatura, token expirando, 24 h, `human_agent`, 429). **A validação ao vivo
com contas reais de terceiros depende do App Review**; antes disso só contas com papel no app.

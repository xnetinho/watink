# Design

Depende de `add-channel-abstraction`. **Fonte e limite:** leitura anterior (navegador headless) da TikTok Business
Messaging API v1.3 em out/2026, **não reconfirmada nesta rodada**: a página não renderiza para `WebFetch`. Tudo abaixo é
**provisório até a tarefa 0.2**.

## Decisão de partida: viabilidade primeiro

Diferente dos outros canais, aqui **o gargalo não é técnico, é de acesso**. O desenho abaixo só vale se a revisão da
TikTok aprovar o Watink como integrador. Por isso a change é **faseada com um portão**:

- **Fase A (sem aprovação):** apenas o adaptador contra um **servidor TikTok simulado**, que valida a interface e as
  regras de resposta. Não toca a rede real.
- **Portão:** aprovação da revisão e acesso ao ambiente real.
- **Fase B (com aprovação):** conexão real, webhook real, validação ao vivo.

## Conexão

OAuth por conta Business (`auth_code` → `account access token`). `credentialsEnc = {access_token, refresh_token,
expires_at, businessId}`. **Não confirmado:** validade do token de mensagens (a Accounts API usa 1 dia com refresh em
`/tt_user/oauth2/refresh_token/`).

## Capacidades

```go
Capabilities{ InitiateConversation:false,
  ReplyWindow: 48h (com limites de mensagens por conversa),
  MaxTextBytes:0 /* nao confirmado */, Media:[image /* so entre paises suportados */],
  Groups:false, Typing:true, ReadReceipts:true, Auth:oauth_review, HasPhone:false }
```

## Regra de resposta (a mais complexa dos cinco)

Para conversa **sem seguimento mútuo**, o limite é por **contagem de mensagens dentro de janelas**, não só por tempo:

| Situação | Pode enviar |
|---|---|
| 48 h após a 1ª mensagem do usuário | até **10** mensagens |
| 48 h após **cada** resposta do usuário | **ilimitado** |
| Usuário ficou inativo > 48 h | **3** mensagens extras |
| Seguimento mútuo | regras normais **[não confirmado em detalhe]** |

`CanReply` precisa de `(lastInboundAt, mensagens enviadas desde então, seguimento mútuo)`: **estado por conversa**, o que
o `Capabilities` simples não expressa. Esta change valida se `ReplyPolicy` precisa virar uma **função da capacidade**
(`func(conversation) Policy`) em vez de só dados; o resultado volta como ajuste do `add-channel-abstraction`.

## Entrega

Webhook (Webhooks API) com `conversation_id`; leitura de reparo por `/business/message/conversation/list/` (até 100) e
`/business/message/content/list/` (últimas 20). **Assinatura do webhook: não confirmada.**

## Pontos de atenção

- **Regiões:** EEE, Suíça e Reino Unido indisponíveis; EUA só com a revisão USDS adicional; resto (incl. Brasil) após DSPR.
- **Imagem:** `media_id` válido por 30 dias, e só quando remetente e destinatário estão em países que suportam imagem.
- **Cartões de pergunta e resposta** e **post do TikTok** como tipos de mensagem de saída: fora da primeira entrega.
- Sem comentários: a API de DM não cobre comentários de vídeo.

## Verificação

Fase A: contrato + servidor TikTok simulado. Fase B: depende inteiramente da aprovação. **Sem aprovação não há
validação ao vivo possível.**

# Proposal

## Why

O Instagram é um dos principais canais de **primeiro contato** de pequenas e médias empresas no Brasil: o cliente manda
DM para tirar dúvida de produto ou preço, e hoje o atendimento acontece no app do Instagram, sem fila, sem histórico
unificado e sem fluxo. A **Instagram Messaging API** (Meta) é oficial, madura e entrega por webhook, o que permite
atender dentro do Watink sem risco de ban.

> **Honestidade sobre a fonte:** as páginas da Meta não carregaram nesta rodada (HTTP 400 em `WebFetch`). Os fatos abaixo
> vêm de uma leitura anterior por navegador headless e **não foram reconfirmados** agora. Antes de implementar, a
> tarefa 0.1 relê a documentação atual, porque a Meta muda versão e política com frequência.

## What Changes

- Canal `instagram` como adaptador de `domain.Channel`, na rota **Instagram API com Instagram Login** (sem exigir
  Página do Facebook), com a rota via Messenger Platform como alternativa.
- **Conexão por OAuth** (Business Login for Instagram): permissões `instagram_business_basic` e
  `instagram_business_manage_messages`; token de longa duração (60 dias) com **renovação automática**.
- **Recepção por webhook** (`messages`, `messaging_postbacks`, `messaging_seen`, `messaging_reactions`,
  `messaging_referrals`) com verificação de assinatura `X-Hub-Signature-256` e desafio de verificação (`hub.challenge`).
- **Envio**: texto (até 1000 bytes), imagem, áudio, vídeo, PDF, reações, **respostas rápidas** e templates; **respostas
  privadas** a comentários (uma por comentário).
- **Regra de 24 h**: só se responde dentro de 24 h da última mensagem do usuário; a **tag `human_agent`** estende para
  **7 dias** para atendimento humano. Modelada como `ReplyWindow` + `ReplyWindowExtension`.
- Identidade do contato pelo **IGSID**; **sem telefone**.

### Fora do escopo
- Publicar posts, stories e moderar comentários; insights; anúncios.
- Iniciar conversa (a API só responde a quem escreveu antes).
- Contas **pessoais** (a API exige conta profissional: Business ou Creator).
- Grupos (não existem no Instagram DM).
- Automação de comentários em massa.

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| **App Review e Business Verification são obrigatórios** para atender contas de terceiros (Advanced Access); prazo **não informado** pela Meta | **Alto (bloqueia a entrega)** | Tarefa 0.2: **iniciar o processo no dia 1**, em paralelo à implementação; o ambiente de desenvolvimento (Standard Access) já permite testar com contas com papel no app |
| Janela de **24 h** (+7 dias com `human_agent`) | Alto | `ReplyWindow` no servidor, contagem regressiva no composer, alerta de SLA; usar a tag `human_agent` só para atendimento humano real (política da Meta) |
| Token de 60 dias precisa de renovação | Médio | Job de refresh antes do vencimento; `status=reauth` e aviso quando falhar |
| Versão da Graph API muda (v25.0 hoje) | Médio | Versão **configurável**; teste de contrato que roda contra a simulação por versão |
| Política de divulgação de bot (obrigatória em algumas jurisdições) | Médio | Aviso na conexão quando houver automação (Assistente/Flow) respondendo |
| Mensagens inativas 30 dias na pasta "Requests" somem da API | Baixo | Documentar; não há como recuperar |
| Limites: 100 chamadas/s por conta (texto), 10/s (áudio/vídeo), 750/h em respostas privadas | Médio | Token-bucket por conta |
| Sem telefone nem e-mail do contato | Baixo | Documentar; o CRM vincula manualmente |

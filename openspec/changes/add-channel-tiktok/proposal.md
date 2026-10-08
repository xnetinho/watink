# Proposal

## Why

O TikTok virou canal de descoberta e de venda no Brasil, e perfis comerciais recebem DMs. A **TikTok Business Messaging
API** permite que uma conta Business troque mensagens diretas por uma plataforma integrada. Trazer isso para o Watink
daria fila e histórico ao atendimento que hoje acontece só no app.

> **Esta é a integração mais incerta das cinco. Leia antes de aprovar.** A página oficial da documentação não renderiza
> para ferramentas de leitura (`WebFetch` devolve só o título), então os fatos abaixo vêm de **uma única leitura anterior
> por navegador headless**, **não reconfirmada**. A API está em **Open Beta** e exige **revisão manual de segurança e
> privacidade**. A primeira tarefa desta change é **provar que a integração é viável para o Watink**, antes de qualquer código.

## What Changes (condicionado ao resultado da tarefa 0)

- Canal `tiktok` como adaptador de `domain.Channel` sobre a Business Messaging API.
- **Conexão por OAuth** da conta Business (código de autorização → token de conta); renovação (tokens da Accounts API
  duram 1 dia e se renovam; o regime do token de mensagens **não está confirmado**).
- **Recepção por webhook** (Webhooks API) e leitura por `conversation/list` e `content/list` (reparo).
- **Envio** de texto, imagem (por `media_id`, válido 30 dias, só entre países que suportam imagem em DM), cartões de
  pergunta e resposta, e ações de "digitando" e "lido".
- **Regras de resposta**: só se responde a quem escreveu antes; sem seguimento mútuo há **limites por conversa** (10
  mensagens em 48 h após a 1ª do usuário; ilimitado em 48 h após cada resposta dele; 3 extras se ele ficar inativo).
- Identidade pelo `conversation_id`; **sem telefone**.

### Fora do escopo
- **Comentários** de vídeos (outra API), publicação de vídeos, anúncios e **TikTok Shop**.
- Iniciar conversa (a API é reativa; só existem links `tiktok.me/{usuario}` e anúncios de DM que abrem conversas).
- Contas pessoais e regiões indisponíveis (EEE, Suíça e Reino Unido não são suportados; EUA exigem revisão adicional).
- Qualquer coisa além de DM de texto/imagem na primeira entrega.

## Riscos

| Item | Risco | Mitigação |
|---|---|---|
| **A revisão (DSPR) pode ser negada ou demorar**: revisão **manual** de segurança e privacidade, sem acompanhamento de status; a TikTok promete só **iniciar** a revisão em 10 dias úteis; recomenda anexar ISO 27001, SOC 2 ou teste de intrusão | **Muito alto (pode inviabilizar)** | Tarefa 0.1: **submeter o formulário já** e tratar a aprovação como pré-requisito; **não implementar além da simulação antes da resposta** |
| **Beta aberto**: contrato pode mudar, e o Brasil aparece em LATAM, mas a disponibilidade para o **nosso tipo de integrador** não foi confirmada | Alto | Confirmar com a TikTok e com uma conta de teste antes de investir |
| **Documentação inacessível** a ferramentas automáticas e não reconfirmada | Alto | Reler pelo navegador na tarefa 0.2 e registrar a versão e a data da leitura |
| Limites por conversa (10/48 h sem seguimento mútuo) quebram o fluxo de atendimento normal | Alto | `ReplyWindow` e contador de mensagens restantes no servidor e no composer; alerta ao atendente |
| Limites globais do app (nível Basic: 10 QPS, 600 QPM) | Médio | Token-bucket global; pedir nível acima só depois de aprovado |
| Imagem em DM depende dos países de ambos os lados | Médio | Capacidade `Media` dependente do par de países; recusar com motivo |
| Custo por mensagem **não confirmado** | Médio | Descobrir antes de oferecer ao cliente final |
| Telefone e e-mail não existem; contas sem seguimento mútuo | Baixo | Documentar; vínculo manual com o CRM |

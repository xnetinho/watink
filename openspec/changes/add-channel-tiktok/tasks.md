# Tasks

> Depende de `add-channel-abstraction` fases 0 a 4. **Há um portão de aprovação da TikTok entre a fase A e a fase B.**

## 0. Viabilidade (antes de qualquer código)
- [ ] 0.1 **Submeter o formulário de revisão (DSPR)** da TikTok e preparar os anexos recomendados (ISO 27001, SOC 2 ou teste
      de intrusão, se existirem); registrar a data e acompanhar
- [ ] 0.2 **Reler a documentação pelo navegador** (a página não renderiza para ferramentas automáticas) e registrar a
      versão e a data: confirmar token, assinatura do webhook, limites, regime de seguimento mútuo e custo
- [ ] 0.3 Confirmar com a TikTok que um **SaaS multiempresa** é elegível (cada cliente autoriza a própria conta Business)
- [ ] 0.4 Decidir, com a resposta, se a change prossegue, se adia ou se é cancelada

## A. Sem aprovação (servidor simulado)
- [ ] A.1 Cliente HTTP e servidor TikTok simulado cobrindo OAuth, webhook, `conversation/list`, `content/list`, envio e 429
- [ ] A.2 Adaptador `tiktok` (`ParseInbound`, `Send` texto/imagem, `Connect` OAuth) e `Capabilities`
- [ ] A.3 `ReplyPolicy` por conversa (10/48 h, ilimitado após resposta, 3 extras): decidir se vira função da capacidade
- [ ] A.4 Dedup por `message_id`, reparo por listagem, token-bucket global (10 QPS / 600 QPM) por app

## B. Com aprovação (portão: resposta positiva da TikTok)
- [ ] B.1 Conexão e webhook reais; validar assinatura do webhook
- [ ] B.2 Validação ao vivo com uma conta Business de teste no Brasil
- [ ] B.3 Pedir nível de rate limit acima do Basic, se o volume exigir

## 2. Frontend
- [ ] 2.1 Botão "Conectar com TikTok" + estado "aguardando aprovação"
- [ ] 2.2 Composer com contador de mensagens restantes na conversa, janela de 48 h e motivo do bloqueio
- [ ] 2.3 Badge e token `channel-tiktok`; traduções; aviso das regiões sem suporte

## 3. Verificação
- [ ] 3.1 Contrato + simulação (fase A); mutação nos testes de segurança
- [ ] 3.2 Validação ao vivo (fase B) e `/qa-analyst`

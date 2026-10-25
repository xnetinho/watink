# ADR 0037: Canal Fake de Teste e Contrato de Extensibilidade de Canais

## Status
Aceito

## Contexto
Na reformulação arquitetural do Watink (consolidate-platform-foundation), a plataforma foi desacoplada do modelo histórico centrado exclusivamente em WhatsApp.
Para garantir e auditar que novos canais de mensageria possam ser adicionados sem alterar o núcleo (core) da aplicação, foi necessária a definição de uma suíte de conformidade executável e de uma implementação de referência mínima (`fake`).

## Decisão
1. **Contrato Universal de Canal (`domain.Channel`)**:
   Todo adaptador de canal implementa:
   - `Kind() domain.ChannelKind`: identificador tipado do canal.
   - `Capabilities() domain.Capabilities`: declaração explícita de recursos (janela de resposta, mídias suportadas, limite de caracteres, suporte a voz/vídeo, digitação, etc.).
   - `Connect(ctx, conn, input)` / `Disconnect(ctx, conn)`: ciclo de vida de conexão.
   - `Send(ctx, conn, msg)`: envio de mensagens de saída, com retorno de `domain.SendResult`.
   - `ResolveAddress(ctx, conn, contact)`: resolução de endereçamento neutro no canal.
   - `ParseInbound(ctx, conn, raw, headers)`: tradução de webhooks/atualizações para `[]domain.InboundMessage`.

2. **Canal Fake (`business/internal/channels/fake`)**:
   Implementa a interface completa sem dependências externas (sem rede real), servindo como prova executável de agnosticismo de canais.

3. **Checklist Automatizado (`channel_checklist_test.go`)**:
   Um teste de conformidade percorre o contrato contra o canal fake e os adaptadores de produção, falhando caso um método não cumpra as garantias estabelecidas.

## Consequências
- A inclusão de novos canais (e.g. Instagram, Mercado Livre, OLX, TikTok) requer apenas a implementação do adaptador correspondente no registro `ChannelRegistry`, sem alterar tabelas centrais, rotas ou controllers de mensageria.
EOF

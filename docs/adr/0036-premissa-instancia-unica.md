# ADR 0036: Premissa Arquitetural de Instância Única

## Status
Aceito

## Contexto
Durante a consolidação da fundação da plataforma (consolidate-platform-foundation), foi estabelecida a premissa de implantação de uma instância única para os seguintes componentes:
1. `watink-engine` (motor Go do WhatsApp com whatsmeow).
2. Poller de canais externos (como o long-polling do Telegram).
3. Job de migrações (`watink-business migrate`).

## Decisão
Fica documentado como decisão explícita de arquitetura que estes serviços operam sob o modelo de **instância única (Active-Passive ou Single Active)** por ambiente:
- O `watink-engine` gerencia sessões em memória e conexão persistente com o WhatsApp via WebSockets; múltiplas instâncias concorrentes para o mesmo tenant/sessão causariam desconexões em cascata.
- O poller do Telegram consome atualizações sequencialmente; múltiplos processos gerariam respostas duplicadas ou conflito de offset sem um mecanismo distribuído de partição.
- O migrador utiliza `pg_advisory_lock` e deve rodar apenas uma vez como tarefa efêmera pré-boot do backend (`watink-migrate`).

## Consequências
- A escala do backend Go de atendimento (`watink-business`) pode ser horizontal, mas o engine e poller permanecem com réplica única (`replicas: 1`).
- Caso no futuro seja necessária alta disponibilidade do engine, deverá ser implementado sharding explícito de conexões baseado em hashing estável de `(tenantId, connectionId)` coordenado via Redis ou líder único.

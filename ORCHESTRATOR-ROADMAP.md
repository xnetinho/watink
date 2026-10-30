# Roadmap do Projeto

## Epics (Visão Estratégica)

- [x] **Epic 1**: Transição do Sistema de Design (MUI v4 → Tailwind + shadcn/ui) | ✅ Concluída
- [x] **Epic 3**: Migração Total Frontend (163 JS/JSX → TSX, remoção MUI, design tokens, lint) | ✅ Concluída jun/2026
- [x] **Epic 2**: Refatoração DI & organização de pacotes (Backend Go) | ✅ Concluída
- [x] **Epic 4**: Qualidade residual — 56 `no-explicit-any` eliminados em 21 arquivos (GAP-ANY) | ✅ Concluída jun/2026 PR #166
- [x] **Epic 5**: Decomposição de God-Files Frontend (GAP-Q/GAP-S) — 20+ god-files eliminados | ✅ Concluída jun/2026
- [x] **Epic 6**: Cobertura de Testes Frontend (GAP-R/GAP-T) — 65/65 testes verdes | ✅ Concluída jun/2026
- [ ] **Epic 7**: Plugin "Assistentes de IA" (Assistant) — 4 modos (pipeline/flow/persona/router), AiGateway, Flow sintético + extensão SDK | 🔧 Planejada — ver `docs/agents/assistants.md`, ADR 0027, plano em `C:\Users\ronaldo\.claude\plans\grill-feature-with-docs-vamos-planejar-memoized-key.md` PR #100

## Milestones

| ID | Descrição | Status |
|---|---|---|
| M1 | Auditoria do Design System | ✅ |
| M2 | Alinhamento de terminologia (CONTEXT.md) | ✅ |
| M3 | Substituição de componentes legados (BaseCard, MetricCard, StatusChip, PageLayout) | ✅ |
| M4 | Migração estrutural (MainLayout, Dashboard, Tickets, Contacts, FlowBuilder) | ✅ |
| M5 | Token Format Migration (JS → CSS Custom Properties + HSL cru) | ✅ |
| M6 | Migração 46 componentes compartilhados MUI → shadcn+TSX | ✅ |
| M7 | Sync Documentação Design System | ✅ |
| M8 | Migração 66+ páginas/modais MUI → shadcn+TSX | ✅ |
| M13 | Ticket Queue Visibility — filtros isGroup + withUnreadMessages + type badges | ✅ PR #98 jun/2026 |
| M14 | GAP-MEDIA-1: mídias WhatsApp salvas em disco, MediaUrl preenchido (backend Go) | ✅ PR #163 jun/2026 |
| M15 | GAP-MEDIA-2: player de áudio customizado com progresso visual e seek | ✅ PR #180 jun/2026 |
| M16 | GAP-ANY: 56 no-explicit-any eliminados em 21 arquivos frontend | ✅ PR #166 jun/2026 |
| M17 | GAP-EIO: Vite watcher resiliente a crash WSL2/EIO | ✅ PR #165 jun/2026 |
| M9 | Storybook + A11y + Visual Regression CI | ⏳ Pendente |
| M18 | GAP-VAL-3: ValidateStringField em ticket_mutation.go (2 endpoints com ShouldBindJSON sem validação de string) | ✅ Concluída |
| M10 | Remoção final MUI v4 (npm uninstall + zero imports) | ✅ jun/2026 |
| M11 | ESLint governance — 298 → 70 erros (-77%) | ✅ jun/2026 |
| M12 | Docs cleanup — CLAUDE.md, docs/dev/, docs/ legado | ✅ jun/2026 |

## Roadmap de produto (próximos épicos)

- [ ] **Epic 8**: Crawl de site na Base de Conhecimento: indexar um site inteiro a partir de uma URL (sitemap ou BFS no
  mesmo domínio). O núcleo já existe e é testado (`business/internal/knowledge/fetch_url_crawl.go`, `CrawlSite`), mas
  não está ligado ao worker nem à UI. Falta: uma Source filha por página (citação por URL), opção na UI com
  `maxPages`/`maxDepth` e cota por tenant, `robots.txt` e intervalo entre requisições, filtro de domínio também no
  sitemap e suporte a `sitemapindex`. | 📋 Planejada: ver "Crawl de site" em `docs/agents/knowledge-base.md`

- [ ] **Epic 9**: Reavaliar código do engine que só testes usam (follow-up de `purge-dead-code`). ~15 funções de `voip/` e
  `calls/` só são chamadas por testes (`IsVideoFrame`, `DecodeVideoFrame`, `PackageH264NALU`/`STAPA`, `NewVideoRtpStream`,
  `BuildVideoAck`, `OfferHasVideo`, partes de `mlow/rangecoder.go`, `ParseStunResponse`, ...) e ~15 não têm uso nenhum
  (`mlow/logging.go`, `signaling/callkey.go` `padRandomMax16`, `signaling_build.go` `CreateCallAck`/`BuildTransportStanza`,
  `stun.go` `FormatStunResponse`/`ClassifyPacket`, ...). São portagem de chamadas de voz/vídeo, **não lixo**: só retomar
  depois que `add-whatsapp-video-calls` fechar, decidindo caso a caso entre ligar (fases 2 a 5) e remover. | ⏳ Bloqueada
  por `add-whatsapp-video-calls`

- [ ] **Epic 10**: Canais agnósticos: Telegram, Mercado Livre, OLX, Instagram e TikTok além do WhatsApp (ADR 0033, planejado).
  Fundação: `add-channel-abstraction` (interface `Channel`, `ContactIdentity`, `conversationKey`, `ChannelConnection`, entrada
  única, regra de resposta no servidor, UX de conexões/composer; **aditiva, o WhatsApp não regride**). Depois, uma change por
  canal, na ordem **Telegram → Mercado Livre → Instagram → OLX → TikTok**: `add-channel-telegram`, `add-channel-mercadolivre`,
  `add-channel-instagram`, `add-channel-olx`, `add-channel-tiktok`. **Aprovações externas devem começar já** (Instagram App
  Review + Business Verification; TikTok DSPR; OLX registro de aplicação), porque definem o prazo. | 📋 Planejada

- [x] **Epic 11**: Visualização de mídia de mensagem de visualização única, sob ação explícita do operador. Hoje o WhatsApp
  entrega aos aparelhos vinculados só um marcador indisponível; o CRM registra o aviso, mas não recebe a mídia (ver
  `engine-go/internal/whatsapp/events_unavailable.go`). Investigar primeiro o caminho assistido pelo celular principal:
  antes de abrir a mídia, o operador responde/cita a mensagem nesse aparelho; se a mensagem sincronizada trouxer metadados
  completos de mídia, o core poderá usar o download padrão do WhatsApp e exibir o arquivo no ticket. **Hipótese, não garantia**:
  resposta automática pelo sistema, miniatura, identificador citado ou simples reconexão não provam que o conteúdo esteja
  disponível. Só habilitar visualização se houver payload completo e download íntegro; nunca tentar recuperar por ID ou
  miniatura isolados. Não abrir nem baixar automaticamente: a ação manual do operador no celular é o opt-in; no CRM, exibir
  somente após ação explícita, respeitando as regras de retenção e acesso à mídia. Não configurar identidade Android/Business
  na sessão vinculada nesta fase: isso exige novo pareamento, altera a identidade de plataforma e traz risco de compatibilidade
  e de política sem resolver a incerteza do fluxo principal. Validar em aparelho e conta de teste, sem mídia real de clientes;
  confirmar sincronização, campos disponíveis e download completo antes de desenhar API/UI. | ✅ Recepção entregue: ao responder no celular principal, o Watink baixa a mídia (hashes e tamanho conferidos), guarda e mostra como mídia comum com selo "Visualização única". **Fora deste Epic (próxima fase, a desenhar):** enviar mídia como visualização única pelo painel de arrastar e soltar; exige teste real, pois o aparelho vinculado pode não ter esse comportamento.

- [ ] **Epic 12**: Pareamento de WhatsApp por código (pairing code) via whatsmeow, sem leitura de QR Code. O backend já
  aceita `usePairingCode`/`phoneNumber` em `POST /whatsappsession/:id` e existe `PairingCodeModal`
  (`frontend/src/components/PairingCodeModal/`), mas o modal está **órfão**: nada no código abre
  `pairingModalOpen`/`setPairingModalOpen(true)` — hoje o único caminho de pareamento por código é
  `PhoneInputDialog` na tela de detalhe (`/connections/:id`), que usa seu próprio estado local
  (`showPairingInput`/`PairingCodePanel`), não o `PairingCodeModal`. Ligar (ou remover) o `PairingCodeModal` depende de
  decidir qual dos dois componentes é o caminho oficial — não fazer os dois coexistirem fazendo a mesma coisa de formas
  diferentes (mesmo tipo de redundância do Epic da tela de Conexões). Expor a opção "Conectar por código" também na tela
  de **lista** (`/connections`), que hoje só tem "Conectar" (QR). | 📋 Planejada

## Próximo Epic — Backend Go DI & Packages

**Branch**: `refactor/backend-di-packages`

Objetivos:
- Garantir DI pura em todos os controllers e services (`business/`)
- Separar camadas: `domain/`, `application/`, `infrastructure/`
- Definir interfaces explícitas entre `business/` e `engine-go/`
- Cobrir endpoints críticos com testes de integração (`httptest`)

## Histórico de Decisões

| Data | Decisão |
|---|---|
| 2026-06-17 | API Docs: Swagger UI → Scalar + swaggo/swag (50+ rotas anotadas, JSON gerado automaticamente) |
| jun/2026 | Nova nomenclatura de branches baseada em Conventional Commits (feat/, fix/, refactor/, chore/, docs/, hotfix/) |
| jun/2026 | docs/dev/ recriado — removido conteúdo Docker Swarm legacy e referências a agentes obsoletos |
| jun/2026 | Epic 3 concluída: MUI v4 removido, 163 arquivos migrados para TSX, design token system 3 camadas |
| jun/2026 | CLAUDE.md reduzido com ponteiros para docs/ especializados |
| 2026-06-16 | Epic 4F: npm uninstall @material-ui/* — zero imports MUI em src/ |
| 2026-06-16 | Epic 4E: ESLint expandido, typescript@5.9.3, 298 → 70 erros |
| 2026-06-14 | Epic 4D: todas as páginas e modais migrados para shadcn/ui + TSX |
| 2026-06-13 | Epic 4B: 46 componentes compartilhados migrados |
| 2026-06-13 | Epic 4A: tokens migrados para CSS Custom Properties em HSL cru |

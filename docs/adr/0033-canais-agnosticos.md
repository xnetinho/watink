# ADR 0033 — Canais agnósticos (Telegram, Mercado Livre, OLX, Instagram, TikTok além do WhatsApp)

**Status:** Proposed (planejado; **nada implementado**, ver `openspec/changes/add-channel-abstraction` e as 5 changes
`add-channel-*`)
**Data:** 2026-10-08

## Contexto

O Watink é WhatsApp-cêntrico: 318 arquivos Go e 141 TS/TSX citam "whatsapp"; o contato é identificado por `number`/`lid`
(vocabulário de JID); a conexão é a tabela `Whatsapps`; o engine fala `wbot.<tenant>.<session>.<cmd>`. O produto passa a
precisar de **Telegram, Mercado Livre, OLX, Instagram e TikTok**.

O precedente mais próximo é o **izapia** (segundo "engine" de WhatsApp). Funcionou porque *continua sendo WhatsApp*: reaproveitou
JID, `FindOrCreate(number, lid)` e o pipeline de entrada, ao custo de colunas próprias em `Whatsapps` e de um
`if EngineType != "whatsmeow"` repetido em 5 pontos de envio. **Um canal sem JID não se encaixa nesse caminho.**

O ADR 0014 já tornou a **saída** plugável (`flow.OutboundChannelAdapter`, `ChannelRegistry`), mas só o WhatsApp foi
registrado. Falta a **entrada** e o **modelo de dados**.

### O que a documentação oficial das plataformas mostrou (out/2026)

Cinco dos seis canais **restringem** quem pode escrever e quando (só o usuário inicia; janela de 24 h no Instagram;
48 h úteis no Mercado Livre; limites por conversa no TikTok). O telefone só é garantido no WhatsApp e na OLX. No Mercado
Livre a conversa é de um **pedido** e na OLX de um **anúncio**, não de uma pessoa. Detalhes e a tabela completa em
`openspec/changes/add-channel-abstraction/design.md`.

## Decisão

1. **Interface `Channel`** pequena (`Connect`, `Disconnect`, `Send`, `ParseInbound`) com **capacidades declaradas em dado**
   (`InitiateConversation`, `ReplyWindow`, `MaxTextBytes`, `Media`, `Groups`, `Auth`, `HasPhone`...). Recursos que só alguns
   canais têm continuam **interfaces opcionais por type-assertion** (padrão atual: `RichMessageEngine`, `GroupEngine`).
2. **`ContactIdentity(tenant, channel, externalId)`** no lugar de `number`/`lid` como identidade. `Contact` continua sendo
   a pessoa; vínculo entre canais é **manual** (mesmo princípio do ADR 0023).
3. **`conversationKey`** no `Ticket`: o contato na maioria dos canais, o **pedido** no Mercado Livre, o **anúncio** na OLX.
   Ticket aberto = `(tenant, conexão, conversationKey)`.
4. **`ChannelConnection`** generaliza `Whatsapp` (`kind`, nome único **por tenant**, credenciais cifradas, `config`).
5. **Entrada única** (`ChannelInbound.Receive`) com dedup por `(conexão, mensagem externa)` e webhooks autenticados por
   segredo por conexão. O consumidor AMQP do WhatsApp vira um produtor entre outros.
6. **Regra de resposta aplicada no servidor** (`CanReply`/`replyPolicy`) e devolvida à UI com o motivo; o composer reflete
   as capacidades.
7. **Expandir e contrair:** tudo aditivo, com backfill e leitura dupla; o WhatsApp só migra para o adaptador depois de
   paridade provada, e a remoção do legado é uma change própria.
8. **Adaptadores dentro do `business`**, não um microsserviço por canal: Telegram, ML, OLX, Instagram e TikTok são
   HTTP + webhook. O `engine-go` continua sendo só o do WhatsApp (conexão persistente, UDP, sessão).

## Opinião sobre a implementação (registro honesto)

- **A abstração é correta e necessária**, mas o risco está na **migração do WhatsApp**, não nos canais novos. Por isso a
  ordem é: fundação aditiva → WhatsApp como adaptador com paridade → integrações.
- **A interface deve nascer com dois canais opostos em mente** (Telegram, aberto; Mercado Livre, restrito) para não
  ficar sob medida para um. A tabela de seis canais foi o teste; o **TikTok já indica** que `ReplyPolicy` precisa ser uma
  função do estado da conversa, não só dado (limite de mensagens por janela).
- **A maior ameaça a prazo não é código, é acesso:** Instagram (App Review + Business Verification), TikTok (revisão
  manual de segurança em beta) e OLX (registro de aplicação não confirmado) dependem de aprovação externa. Elas devem
  **começar já**, em paralelo.
- **Ordem recomendada:** Telegram → Mercado Livre → Instagram → OLX → TikTok. Telegram valida a interface sem burocracia;
  ML tem o maior valor comercial; TikTok é o mais incerto e fica por último, com portão de viabilidade.

## Consequências

- **Positivas:** o núcleo deixa de saber o canal; novo canal vira um adaptador; UX consistente (catálogo, composer guiado
  por capacidades, badge); regras restritivas de plataforma viram produto visível em vez de erro depois do envio.
- **Negativas / custo:** migração de dados sensível (identidade do contato, chave do ticket aberto, fila de visibilidade
  de tickets em `pkg/auth/tenant.go`); ~6 pontos de construção de JID a unificar; o glossário atual (`Whatsapp`,
  `Contact` "do WhatsApp", _Avoid: channel_) precisa ser revisto; a fila única e sequencial de eventos (dívida do
  CLAUDE.md) piora com canais de marketplace de volume.
- **Riscos de produto:** cada plataforma pode mudar política (Mercado Livre mudou a mensageria em 02/02/2026 com agentes de
  IA); dependência de aprovações externas pode atrasar ou inviabilizar canais.

## Pendências (honestas)

- Documentação da Meta (Instagram) e da TikTok **não foi reconfirmada** nesta rodada (as páginas não carregam para
  leitura automática); as changes têm uma tarefa 0 de releitura.
- OLX: **acesso aberto a SaaS não confirmado** (`client_id` é fornecido no registro da aplicação).
- Mercado Livre: aprovação do app, validade do token e dados do comprador em `/questions` **não confirmados**.
- Telegram: `request_contact` (telefone) e requisitos do Business Mode **não verificados**.
- Fila de eventos global: decidir filas por canal/shard antes de ligar canais de volume.

## Termos propostos para o `CONTEXT.md` (a aprovar)

- **Channel**: tipo de canal de mensagens (`whatsapp`, `telegram`, `mercadolivre`, `olx`, `instagram`, `tiktok`) com suas
  capacidades e regras de resposta.
- **ChannelConnection**: conexão persistente de um tenant a um canal (credenciais cifradas, status). Generaliza `Whatsapp`.
- **ContactIdentity**: identidade de um Contact em um canal (`channel`, `externalId`). Um Contact tem N identidades.
- **conversationKey**: o que define "a mesma conversa" em um canal (contato, pedido ou anúncio).
- **Capabilities / ReplyPolicy**: o que o canal permite e quando se pode responder.
- Ajustar: **Contact** deixa de ser "do WhatsApp"; **Whatsapp** passa a ser o caso `kind=whatsapp` de ChannelConnection e
  o aviso _Avoid: channel_ é revisto.

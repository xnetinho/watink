# Decisões e backlog (29/out/2026)

Registro do que foi decidido ao fechar a consolidação de canais, para não depender de memória de sessão.

## Decisões sobre branches e stashes antigos

| Item | Decisão | Motivo |
|---|---|---|
| `feat/theme-auto-mode` (5/out) | **Descartado** | O toggle pessoal (`6d41df7c`, já na `develop`) segue o navegador quando o usuário não escolheu e reage à mudança do sistema. Só faltaria a opção "Automático" nas Configurações da empresa, de pouco uso. Foi mergeado apenas em `test/ghcr-images`, branch longa abolida. |
| `backup/ghcr-before-theme-toggle` | Backup de `test/ghcr-images` | Estado da branch longa antes do toggle. Ver `git_workflow_policy.md` (não há branch longa de teste). |
| `feat/wbot-contract`, `feat/migrate-wbot-contract` | Já na `develop` | PRs #33, #34 e #35. O commit `825afadb` tem o mesmo patch-id do `dcc98340`. |
| `feat/video-send-camera` | Já na `develop` | PR #13. |
| `feat/assistant-audio-gateway-models` (PR #587 do repositório pai) | **Adiado** | Seletor de modelo de transcrição/fala no `AssistantCreator`. O backend já existe (`TranscriptionModel`/`SpeechModel` no AiGateway). Decidir junto com a frente de voz, porque o provedor pode mudar. Conflita com a `develop`. |
| `feat/inventory-page-image` (PR #658 do repositório pai) | **Fora desta frente** | Tela de Estoque do repositório pai, com conflitos. |
| Stashes `{1}`, `{2}`, `{3}` | Equivalente já na `develop` | Comparação linha a linha. |
| Stash `{0}` | Ferramenta de ambiente | Skills/commands do OpenSpec, não é código do produto. |

## Versão do cliente WhatsApp (405 "client outdated")

O WhatsApp recusa versões velhas do cliente e a embutida no `whatsmeow` envelhece (em 3/nov/2026 as duas conexões do `lt` caíram com 405). Decisões do usuário:

- **Go 1.26 e `whatsmeow` de 02/nov** (PR #136); o Go sobe junto porque a biblioteca passou a exigi-lo.
- **Escolha global do superadmin** na tabela própria `WaVersionSettings` (linha única, sem RLS, `SELECT` para o papel do engine): modo `auto` ou `manual` com o número.
- **Automático:** site oficial (`whatsmeow.GetLatestVersion`) → `currentVersion` do registro `wppconnect-team/wa-version` (sem o sufixo `-alpha`) → versão embutida. Fonte menor que a embutida é ignorada. **Manual:** só o número cadastrado, sem consultar a rede (número inválido cai no automático).
- **405:** o engine marca a sessão `DISCONNECTED`, renova a versão e reconecta UMA vez por versão (sem martelar o servidor). Quando o superadmin salva outra versão, as sessões recusadas reconectam sozinhas.
- **Tela** (PR seguinte): as quatro versões lado a lado (embutida, oficial, registro, vigente) e o seletor automático/manual. O business lê o engine pela API interna do `groupsapi` (`GET /wa-version`, `POST /wa-version/refresh`). Essa API não tem token: o `GROUPS_API_TOKEN` foi removido de todo o sistema em 03/nov/2026 (a defesa é a rede: porta só `expose`, nunca `ports:`, como no RabbitMQ e no áudio das chamadas, ADR 0031).

## Biblioteca de figurinhas

Como o WhatsApp Web guarda figurinhas (investigado em 3/nov/2026, sem implementar): as **favoritas** vivem no app state da coleção `regular_low`, índice `favoriteSticker` (`StickerAction`: URL, `directPath`, `mediaKey`, `fileEncSHA256`, tamanho, `isFavorite`, `imageHash`), e as **recentes** no History Sync (`RecentStickers`/`FavoriteStickers`). Cada conta tem a sua lista; não existe lista da empresa no protocolo.

Decisões do usuário: a biblioteca é da **EMPRESA** (todos os atendentes e todas as conexões WhatsApp veem as mesmas), **sem limite** de itens, **qualquer atendente** salva e remove (permissão `tickets:update`), e **regra A de espelhamento**: nada é escrito no celular de ninguém.

Aviso ao salvar: `POST /stickers` devolve `alreadySaved` (por empresa) e a tela diz "já está na biblioteca" em vez de "salva" quando a figurinha já existia.

Entregue (PR da biblioteca): tabela `StickerLibrary` (RLS por empresa, índice único por empresa e hash do conteúdo, então salvar de novo não duplica), `GET/POST/DELETE /stickers`, envio por `stickerId` (id da biblioteca, nunca URL), "Salvar figurinha" no menu da mensagem (recebida ou enviada) e seletor de figurinhas no compositor, só WhatsApp.

Fica para depois (Parte 2): **ler as favoritas dos celulares** e uni-las à biblioteca. Exige ligar `EmitAppStateEventsOnFullSync` e forçar a ressincronização de `regular_low`, e confirmar se o download de uma favorita funciona sem o `fileSHA256` (o `StickerAction` não o traz; `imageHash` talvez sirva).

## Backlog

Ordem acordada: A, B, E, F (C encerrado, D entregue). G (múltiplas bibliotecas WhatsApp), H (mídia em S3 com limite por tenant) e I (retenção de conversas e análise do Timescale) anotados em 5/nov/2026, sem ordem definida.

**Entregue / encerrado**

- **Vigia de entrega e reenvio manual** (#115/#116): ack 0 parado >2 min vira falha (ack 5); recibo 1–4 depois do 5 desfaz; `UseRetryMessageStore` ligado.
- **Edição de mensagens** (#118–#122): receber e enviar, histórico, selo.
- **Visualização única, recepção** (#124, Epic 11): ao responder no celular principal o Watink baixa a mídia, guarda e mostra como mídia comum com selo.
- **Textos fixos** (#127–#131): `src/` inteiro sem texto fixo em português; guarda `noHardcodedText` sem exceções (valor que não é texto de tela leva `// i18n-ignore`); guarda `noModuleLevelTranslation` impede `tr()` fora de função.
- **Menores (antigo item 5):** WAV (conversão para OGG/Opus), vídeo e PDF já saem como o cliente oficial; emojis locais (#105). Sticker pelo painel entregue (#142) e normalizado (botão "Figurinha" em imagem, GIF e vídeo curto; o engine converte para WebP 512x512 com a escada de qualidade 100 KB estática / 500 KB e 10 s animada, e marca `isAnimated`; só WhatsApp, sem legenda; nos outros canais 422 `sticker_not_supported`). Validado no celular do Diomedes em 3/nov/2026: PNG retangular, PNG transparente e GIF animado chegam como figurinha, a animada anima. O engine devolve o WebP final (`message.media` com `replaced`) e a conversa passa a mostrar o que o destinatário recebeu.
- **GIF enviado como mídia comum** (corrigido): o painel mandava o GIF como `ImageMessage` `image/gif`, que o servidor do WhatsApp aceitava (ack 1) e o celular descartava sem avisar. Agora o engine converte o GIF em MP4 (H.264 baseline, sem áudio, lado maior até 512 px, dimensões pares, corte em 15 s) e envia como `VideoMessage` com `gifPlayback`, como o cliente oficial. A conversa toca o GIF em loop e sem som (enviado ou recebido) e um selo avisa quando o GIF foi cortado. Validado no celular do Diomedes em 3/nov/2026 (GIF de 3 s e de 30 s cortado em 15 s). Limitação: o MP4 não tem transparência, então GIF transparente fica sobre fundo preto.
- **Enviar mídia como visualização única** (B, #134/#139): validado no celular do Diomedes (3/nov/2026). O reenvio pela exclamação mantém a marca (#139).
- **Versão do cliente WhatsApp** (#136–#138) e **remoção do `GROUPS_API_TOKEN`** (#140): ver a seção própria acima.

**Limitação conhecida (não corrigida, decisão do usuário em 3/nov/2026):** com as DUAS pontas sendo conexões do mesmo tenant (empresa conversando com ela mesma, como no teste 1→2), a mensagem recebida pela segunda conexão não é gravada: o ID da mensagem do WhatsApp é o mesmo nas duas pontas e `Messages` tem chave primária só no ID, então a gravação vira "duplicata". Em produção o destinatário é um cliente externo, sem colisão. O Telegram já resolveu o caso equivalente com `ExternalID` + conexão.
- **"Aguardando mensagem" no primeiro envio (antigo item 2): ENCERRADO por falta de evidência** (30/out/2026). O aviso foi visto no celular do Diomedes antes do vigia e do `UseRetryMessageStore`; não reproduzido (texto a contato novo, sessão local apagada e número fora do WhatsApp foram testados e não geram o sintoma). Reabrir se voltar a ocorrer, anotando o horário para cruzar com o log do engine.

**Fila**

1. **A. Datas/horas de entrega e leitura (estilo "Dados da mensagem" do WhatsApp Web).** Hoje só `Messages.ack` é guardado; o engine descarta o horário (`v.Timestamp`) e o remetente (`v.Sender`) do recibo. Desenho: tabela `MessageReceipts` (mensagem, participante, estágio entregue/lida/reproduzida, horário), só no modelo GORM (no `lt`, criar à mão com o DDL idêntico); o engine envia horário e participante no `message.ack`; `GET /message/:id/receipts`; item "Dados da mensagem" no menu da mensagem, textos nos 3 idiomas. **Decisões do usuário:** vale para grupos (lista por participante) e conversa individual; mensagens antigas ficam sem data, sem backfill. Limite: sem recibo de leitura (confirmação desligada no aparelho do contato) só aparece "entregue".
2. **B. Enviar mídia como visualização única** (painel de arrastar e soltar, como no WhatsApp Web): hoje só existe a recepção. Falta botão de olho por mídia, campo `viewOnce` no envio e `ViewOnce=true` em `buildMediaMessage`; aproveitar para expor sticker no painel. Decidir antes como a mensagem aparece para nós depois de enviada e se vale para áudio. Risco: o aparelho vinculado pode não ter esse comportamento; validar com a conexão 2 antes de entregar.
3. **E. Migração `Whatsapps` → `ChannelConnections`.** `models.Whatsapp` em 38 arquivos de produção, `WhatsappID` em 52, tabela `"Whatsapps"` em 21; 63 arquivos de teste e 65 no frontend (`whatsappId`). Fatias verticais, sem coexistência, suíte inteira verde em cada uma. Alto risco de regressão: começa por um desenho das fatias. **Auditoria e plano de fatias em `docs/dev/migracao-whatsapps-para-channelconnections.md` (5/nov/2026):** as tarefas 3.3, 3.4 e 3.7 do `consolidate-platform-foundation` estão marcadas como concluídas, mas o código não as fez (a coluna `whatsappId` existe em 9 tabelas, `Whatsapps` e as 13 rotas `/whatsapp/*` existem, e o repositório de `whatsapp_connection_details` nunca é instanciado). Três decisões do dono pendentes antes da fatia 1 (destino dos ids, onde moram QR/número/presença, e se os recursos de mídia vão para o adaptador agnóstico).
4. **F. Voz** (ver `backlog-voz-transcricao-sintese.md`): transcrição, síntese e chamadas automatizadas pelo flowbuilder; exige estudar os dois repositórios MIT e desacoplar `internal/calls` e `internal/voip`. Primeiro PR: estudo de viabilidade.
5. **G. Múltiplas bibliotecas de conexão WhatsApp** (anotado a pedido do usuário em 5/nov/2026; nada estudado nem implementado). Hoje a única biblioteca é o `whatsmeow` (Go), usado só no `engine-go`; `business` e `contracts` não o importam. Adicionar:
   - **zapo** (https://github.com/vinikjkkj/zapo): implementação TypeScript do protocolo do WhatsApp Web, pacote `zapo-js`, Node >= 20.9, API estável desde a 1.0.0 (SemVer). Multi-sessão e alta escala; stores opcionais (SQLite, Redis, Postgres, MySQL, Mongo); mídia opcional via `@zapo-js/media-utils` (precisa de `ffmpeg` e `ffprobe`); acelerador criptográfico opcional em Rust/WASM.
   - **whatsapp-rust** (https://github.com/oxidezap/whatsapp-rust): biblioteca assíncrona em Rust, crate `whatsapp-rust` (0.7), pareamento por QR e por código, mensagens, mídia, reações, recibos, histórico, grupos, comunidades, canais, status, e chamadas de voz e vídeo (1:1 e em grupo; recurso `voip` opcional). Armazenamento, transporte e runtime substituíveis. Inspirada no `whatsmeow` e no Baileys.
   - Ambas são clientes **não oficiais**: o README da `whatsapp-rust` avisa que o uso pode violar os Termos da Meta e levar à suspensão da conta. O mesmo risco já vale para o `whatsmeow`.

   **O que já existe no código para apoiar isso:** o `business` fala com o engine por RabbitMQ pelo contrato `contracts/wbot` (comandos `session.start`, `message.send.*`, `media.download`, etc., e eventos `message.received`, `message.ack`, `message.media`, ...), mais a API HTTP interna do engine (grupos, comunidades, versão). Um engine novo, em outra linguagem, entra como outro consumidor desse contrato, sem tocar no `business`. A escolha por conexão já tem base: `Whatsapp.EngineType` (padrão `whatsmeow`) e `WhatsAppSessionService.engineFor`, que acha o provedor registrado para o tipo e falha com `nenhum provedor registrado para engineType ...` se não houver. O `whatsmeow` via `engine-go` é hoje o único provedor registrado.

   **Perguntas a decidir antes de desenhar (não decididas):**
   1. A escolha da biblioteca é **por conexão** (um número em `whatsmeow`, outro em `zapo`) ou **por instalação**? Por conexão exige o roteamento por `engineType` até o consumidor certo e uma política de qual engine assume cada sessão.
   2. **Estado de sessão.** Cada biblioteca guarda chaves e sessões no seu formato. Trocar a biblioteca de um número já pareado provavelmente exige **parear de novo** (ler o QR de novo). Não há como migrar a sessão do `whatsmeow`; precisa ser confirmado.
   3. **Paridade de recursos.** O `whatsmeow` já cobre o que o Watink usa (recibos com horário, edição, visualização única, enquetes, mídia com metadados, figurinhas, chamadas de voz, versão do cliente). Cada recurso novo exigiria conferir se a biblioteca o oferece e como; o que faltar vira "não suportado nesta biblioteca", como já é no Telegram.
   4. **Operação.** O engine novo é outro binário (Node ou Rust) com outra imagem, outro consumo de memória e outros logs. A conversão de mídia (`ffmpeg`) hoje vive no `engine-go`; decidir se o engine novo reusa o mesmo código ou tem o seu.
   5. **Teste.** Falta um conjunto de testes do contrato que rode contra **qualquer** engine, para provar a paridade em vez de supô-la. O `wbot_contract_test.go` cobre só a forma das mensagens.

   **Primeiro passo sugerido:** um estudo de viabilidade (como o do item F), por biblioteca, mapeando cada comando e evento do `contracts/wbot` para a API dela e listando o que falta. Sem decidir implementação antes dele.
6. **H. Armazenamento de mídia em S3 (R2, Backblaze etc.) com limite por tenant e rotação automática** (anotado a pedido do usuário em 5/nov/2026; nada implementado).
   - **Pedido:** suporte a storage S3-compatível (Cloudflare R2, Backblaze B2 e similares) para as mídias das conversas, com **limite de tamanho por tenant** e **rotação automática**: as mídias mais recentes apagam as mais antigas quando o tenant passa do limite.
   - **Como está hoje:** as mídias de conversa vão para o **disco local** (`public/media/<sha256>.<ext>`, servido por `r.Static("/public/media", ...)`), gravadas por `mediastore.SaveMediaBase64`/`SaveMediaReader` e lidas por `mediastore.ReadLocal` (Telegram, RabbitMQ, biblioteca de figurinhas). O **caminho não tem o tenant** (é só o hash do conteúdo), então hoje não há como somar o tamanho por tenant nem apagar por tenant. Já existe um object store S3-compatível (`domain.ObjectStore`: `Upload`, `Download`, `PresignedGetURL`, `Delete`, `Describe`), configurado por `S3_*`, mas usado só pela Base de Conhecimento, pelas fotos de checklist e pelas gravações de chamada; `GET /system/storage` só mostra o status.
   - **O que decidir antes de desenhar:**
     1. **Chave com tenant.** Mudar o caminho para `tenant/<id>/<hash>` dá a contabilidade e a exclusão por tenant, mas **quebra a deduplicação por conteúdo entre tenants** (hoje o mesmo arquivo de dois tenants é um só, o que também é um vazamento de existência entre empresas). Ainda precisa decidir o destino das mídias já gravadas (sem migração incremental: ou reprocessar uma vez, ou deixar as antigas no disco).
     2. **O que "apagar a mais antiga" significa para a conversa:** a mensagem fica, mas sem o arquivo. A tela precisa de um estado "mídia removida por limite" (hoje só existe "indisponível"/"pendente"). Figurinhas da biblioteca, anexos de protocolo e gravações de chamada **não podem** ser rotacionados sem aviso (outra mensagem ou a biblioteca aponta para o mesmo arquivo); a rotação precisa respeitar isso.
     3. **Onde fica o limite:** por plano (`Plans`) ou valor por tenant editável pelo superadmin. Medir o consumo exige uma coluna ou tabela de uso por tenant atualizada a cada gravação e remoção.
     4. **Servir o arquivo:** URL assinada com TTL curto (já existe `PresignedGetURL`) em vez de `/public/media` aberto; o frontend usa `getBackendUrl(mediaUrl)`, então a URL passa a ser outra.
     5. **Credenciais por tenant ou globais:** um bucket da instalação com prefixo por tenant, ou o tenant traz o próprio bucket. Muda isolamento, custo e suporte.
     6. **Engine em outro contêiner:** hoje o business empacota os bytes (`mediaData`) ao publicar o comando porque o engine não vê o disco. Com S3 o engine poderia baixar direto pela URL, o que tira o arquivo do RabbitMQ.

7. **I. Retenção de conversas por tenant com rotação automática, e análise de migrar para TimescaleDB** (anotado a pedido do usuário em 5/nov/2026; nada implementado).
   - **Pedido:** limite de tempo para guardar conversas **por tenant** (exemplo: 60 meses), apagando as mais antigas conforme as novas entram. Analisar se vale trocar o banco para Timescale.
   - **Como está hoje:** não existe nenhum job de retenção ou expurgo; `Messages` só cresce. `Messages` não tem chave estrangeira entrando nem saindo, mas várias tabelas apontam para ela **por convenção, sem FK** (`MessageEdits`, `MessageReceipts`, `PollResults`, `group_campaign_*`, `group_watch_matches`, e `Messages.quotedMsgId`), então apagar uma mensagem precisa apagar essas linhas à mão. No `lt` a tabela é minúscula (150 linhas, 208 kB), então **não há medida real de volume**: a análise de desempenho abaixo é estimativa, não medição.
   - **Análise do Timescale (parcialmente verificada):**
     - **Verificado na documentação oficial:** numa hypertable, todo índice único (inclusive a chave primária) **precisa conter a coluna de particionamento (o tempo)**. A chave primária de `Messages` é só `id` (texto): teria de virar `(id, createdAt)`, e `FindByID`, o `ON CONFLICT` de `CreateIfNotExists` (idempotência de reentrega do engine) e todo `WHERE id = ?` perdem a garantia de unicidade só por `id`. Isso mexe no coração do recebimento de mensagens.
     - **Verificado:** a edição **Community** (licença Tiger Data/TSL) é grátis para uso próprio, mas **não pode ser vendida como serviço**; a edição Apache 2 é livre, porém sem os recursos mais novos. O projeto distribui imagens (GHCR) e há o Modo SaaS: é preciso ler a TSL com cuidado antes de empacotar o Timescale numa imagem que terceiros usam.
     - **Não verificado (fonte indisponível na hora; conferir antes de decidir):** como a retenção por chunk (`add_retention_policy`/`drop_chunks`) se comporta; compressão/columnstore; compatibilidade da imagem com **PostGIS e pgvector** (a imagem atual é `postgis/postgis:16-3.4` + `pgvector`, e o Timescale também é uma extensão; combinar as três numa imagem só é trabalho de build); e RLS por tenant em chunks.
     - **O ponto central (raciocínio, não medição):** o Timescale apaga por **chunk de tempo inteiro**, para **toda a tabela**. A retenção pedida é **por tenant** (60 meses para um, outro valor para outro). Uma política única por hypertable não dá isso: ou o chunk só é descartado quando **todos** os tenants o liberam (o tenant de retenção mais longa segura o chunk de todos), ou se volta a `DELETE` por linha, que é o que já se faria sem Timescale. Ou seja, o ganho principal do Timescale (descarte barato de partição) **não se aplica bem a retenção por tenant**.
     - **Alternativas a medir antes:** (a) job de expurgo em lote no próprio Postgres (`DELETE ... WHERE tenantId = ? AND createdAt < ?` em pedaços, com `VACUUM`/autovacuum observado) com índice `(tenantId, createdAt)`; (b) **particionamento nativo por faixa de data** (`PARTITION BY RANGE (createdAt)`), que dá o descarte barato de partição sem a extensão, com a mesma exigência de a chave primária conter a data; (c) Timescale só para `OutboxEvents`/logs, que são séries de tempo de verdade e sem a restrição de unicidade por `id`.
   - **Decisões a tomar antes de desenhar:**
     1. **O que "conversa" inclui ao expurgar:** mensagens, tickets, logs de ticket, recibos, edições, enquetes, embeddings, gravações de chamada. Ticket aberto ou com atividade recente nunca pode perder a mensagem mais antiga que ele ainda mostra.
     2. **Retenção por tenant** guardada onde (tabela de configuração por tenant, ou campo em `Plans`) e quem edita (superadmin). Valor padrão, mínimo e o que fazer ao **reduzir** a retenção de um tenant (apaga de uma vez? em lotes? com aviso?).
     3. **Obrigação legal:** apagar conversas é irreversível; precisa de confirmação explícita, registro de auditoria e, possivelmente, exportação antes. **LGPD**: a retenção também serve de política de minimização, mas o tenant pode precisar do oposto (guarda mínima exigida por lei). Não é decisão técnica.
     4. **Backup:** o expurgo não apaga o que já está nos backups; a política de retenção dos backups precisa acompanhar.
     5. **Ligação com o item H:** apagar a mensagem deve apagar também a mídia no S3 (e vice-versa o limite de mídia deixa a mensagem sem arquivo). Os dois itens devem compartilhar o mesmo conceito de "o que pode ser removido".
   - **Primeiro passo sugerido (sem tocar em código):** medir. Gerar um volume realista de `Messages` num banco descartável (milhões de linhas, vários tenants), e comparar `DELETE` em lote, partição nativa e hypertable, nas três operações que importam: **inserir** (idempotente por `id`), **ler a conversa de um ticket**, e **expurgar por tenant**. Só então decidir se o Timescale paga a complexidade.

- **Bug (6/nov/2026): a notificação de chamada recebida não some da tela quando a chamada é atendida.** Achado pelo dono ligando do celular. A chamada toca ao mesmo tempo no celular e no Watink; ao ser atendida (aqui, pela API), o celular para de tocar, mas o painel do Watink continua com a notificação e só sai se o operador apertar "recusar". **Não tratado**, só registrado. Pistas, sem verificação: (1) o painel escuta `call.incoming` e deveria remover o modal ao receber `call.answered` ou `call.ended`; vale conferir se o evento `call.answered` chega a quem NÃO atendeu (a sala SSE de cada operador) e se o `IncomingCallModal` reage a ele; (2) o atendimento pela API não passa pelo estado da tela, então o modal nunca ouve que a chamada foi assumida; (3) "recusar" depois de atendida deveria ser inofensivo (`ErrAlreadyAnswered`), e vale conferir que ele não derruba a chamada de quem atendeu. Teste a escrever: dois operadores online, um atende, o outro precisa perder o modal sem ação.

  **Verificado em 7/nov/2026: não reproduzido.** Duas rodadas no `lt`, com um monitor SSE (mesmo usuário do painel) que gravou cada evento `call.*`
  que o painel recebe. (1) Chamada do Android pessoal **atendida no Android do número de teste** (WhatsApp Business): o engine recebeu `accept`
  de outro aparelho e `terminate accepted_elsewhere`; o painel recebeu `call.incoming` e `call.ended` (`missed / accepted_elsewhere`); o toque
  sumiu sozinho. (2) Chamada **deixada tocar**: `call.ended timeout` aos 45 s e o toque sumiu. Cada elo tem teste: engine
  (`TestAcceptedElsewhere_EndsWithoutSendingAnything`), business (`TestHandleEnded_AnsweredElsewhereDismissesTheRingForEveryone`, novo) e
  frontend (`callReducer.test.ts`). A causa do que o dono viu em 6/nov continua desconhecida (provavelmente o atalho de teste que atendia pela
  API sem passar pelo estado da tela); não voltou a aparecer.

- **Defeito achado nesse teste e corrigido (7/nov/2026): oferta de chamada duplicada virava `missed/busy`.** O servidor do WhatsApp entregou a mesma
  oferta duas vezes (mesmo `call-id` e mesmo id de stanza, 56 ms de intervalo, a segunda com outra lista de relays). `Session.OnOffer` só olhava
  "há chamada ativa?" e tratava a repetida como outra chamada com a linha ocupada: saía `call.missed busy` e o registro fechava como
  `missed/busy` em 0,06 s, com a chamada ainda tocando. Agora uma oferta cujo `call-id` já é a chamada ativa é ignorada. O whatsmeow não
  deduplica (`handleCallEvent` despacha tudo), então a guarda é nossa. Teste: `TestOffer_DuplicateOfActiveCallIsIgnored` (falha sem a guarda).

- **Origem do chamador em chamadas (achado de 7/nov/2026, ainda não implementado).** O `<offer>` traz `platform` e `version`, e o whatsmeow já os
  entrega em `events.CallOffer.RemotePlatform/RemoteVersion` (o engine hoje os descarta). Valores vistos: `android`, `web` (com dispositivo `:84` no
  JID e versão `2.3000.x`) e `iphone` (não `ios`). O protocolo lista também `smba` (Business Android, visto só em `accept`), `smbi`, `ipad`, `darwin`,
  `uwp`. O ID da chamada tem o mesmo formato (32 hex) em todas as origens, então NÃO serve. Proposta: gravar a plataforma em `CallLogs` e mostrar no
  registro de chamada. Pendente: decisão do dono.

## Webchat: situação e decisão pendente (7/nov/2026)

Anotado a pedido do usuário para **avaliação posterior**. Nada foi implementado nem removido; só o parâmetro de rota e o snippet de
embed foram renomeados de `whatsappId` para `connectionId` (fatia 6 da migração).

**O que é:** widget de chat para o site do cliente. O visitante informa nome, e-mail e telefone e a conversa chega como ticket. Na
tela de Conexões há "Adicionar Webchat", um modal de aparência (cor, ícone, posição, título, subtítulo, campos) e o código de embed.

**O que existe hoje (verificado no código, não é efeito da migração):**
1. `POST /webchat/:connectionId/tickets` e `GET /webchat/:connectionId` são rotas **autenticadas** (plugin `webchat`, permissões
   `tickets:create` e `connections:read`). Um visitante anônimo do site não consegue chamá-las.
2. O snippet de embed carrega `<backend>/public/webchat`, que **não existe** (o backend só serve `/public/media`). O widget não carrega.
3. O modal envia `type: "webchat"` e `chatConfig` (aparência) ao salvar a conexão; o backend **ignora os dois**, então a aparência não
   persiste e a conexão é criada como uma conexão WhatsApp comum.
4. A rota cria contato (por e-mail) e ticket `pending`, mas não grava a mensagem inicial nem avisa o painel (há um comentário
   "Emitting message to RabbitMQ would go here").
5. O plugin está marcado como `pro` e o catálogo lista "Webchat incluso" em planos.

**Decisão a tomar depois (duas saídas):**
- **Terminar:** script/widget público servido pelo backend; rota pública sem login com limite de taxa, validação do `Origin` e proteção
  contra abuso; mensagens em tempo real nos dois sentidos pelo adaptador de canal (`kind` próprio, não WhatsApp); aparência gravada;
  teste de ponta a ponta com um navegador.
- **Remover a casca:** tirar o plugin, o modal e a opção da tela, e o item dos planos.

Não há prioridade definida. Enquanto isso, o Webchat não deve ser anunciado como funcional.

## Aparelho do contato no rodapé do balão (7/nov/2026)

Pedido do dono: cada mensagem **recebida** mostra no rodapé do balão o aparelho do contato, com "~" na frente (estimativa): `~Android`,
`~iPhone`, `~Web`, `~Desktop`.

**O WhatsApp não informa o aparelho na mensagem.** O `MessageInfo` do whatsmeow não tem campo de plataforma (só `ID`, `Sender`, `Chat`,
`PushName`, `Type`, `Category`...). O número do dispositivo no JID separa só "celular" (0) de "vinculado" (>0), e não distingue Android de
iPhone. O único sinal que separa os três é o **formato do id da mensagem**, e é previsão, não contrato.

**Regra adotada** (`frontend/src/lib/senderDevice.ts`, copiada de `getDevice` do Baileys, WhiskeySockets/Baileys, `src/Utils/messages.ts`, MIT,
na mesma ordem): `3A` + 18 caracteres = iPhone; `3E` + 20 = Web; 21 ou 32 caracteres = Android; `3F...` ou 18 caracteres = Desktop; resto = sem selo.

**Validado com mensagens reais (7/nov/2026):** Android (`ACF893AD...`, 32 hex, texto e foto), iPhone (`3AF2FA95...`, 20, texto e foto) e WhatsApp Web
(`3EB0908D...`, 22). As três origens batem com a regra. Amostra pequena (2 mensagens por aparelho, 1 de cada origem de teste).

**Onde aparece:** só mensagem recebida do WhatsApp (não em mensagem nossa, apagada, registro de chamada nem outros canais). Sem coluna nova
nem mudança no engine: a regra usa o id que o painel já recebe. Quando há outro selo no canto esquerdo ("Editada", "Visualização única", "GIF
cortado"), o do aparelho cede o lugar.

**Limites conhecidos:** (1) é heurística: um app atualizado pode mudar o formato sem aviso; (2) mensagens enviadas pelo próprio painel
(`3EB0` + 16 hex) não casam e não ganham selo, o que é o desejado; (3) iPhone e Android antigos, Desktop e mensagens de grupo não foram
testados; (4) um cliente de terceiros que use o prefixo `3EB0` pode aparecer como Web.

## Reação ao registro de chamada (7/nov/2026)

**O registro de chamada TEM suporte a reação no WhatsApp** (o dono verificou no celular). Eu havia afirmado o contrário e abri o #169 para tirar a
faixa de reações do menu; o dono o fez fechar sem merge. A reação não funcionava por outro motivo: **descasamento de id**.

**Prova (diagnóstico no engine, `lt`, 7/nov/2026, 23:32 a 23:33):** o alvo (`key.id`) de uma reação feita no celular a um registro de chamada era
exatamente o `call-id` (`00FC188D...`), em claro (não cifrada). O Watink guarda o registro como `call:<call-id>`. Dois defeitos, um em cada sentido:
(1) **recebida**: `handleMessageReaction` fazia `FindByID("<call-id>")`, não achava `call:<call-id>` e descartava a reação em silêncio; (2) **enviada**:
`ReactToMessage` mandava `targetMsgId = "call:<call-id>"` ao WhatsApp, que não conhece esse id, e o celular não mostrava a reação.

**Correção:** `domain.CallMessageID`, `WhatsAppMessageID` e `WatinkMessageIDs` concentram a convenção. O adaptador manda o `call-id` ao WhatsApp e o evento
de reação tenta o id do WhatsApp e o do registro. Testes: `TestHandleMessageReaction_ReacaoAoRegistroDeChamadaAcha_CallPrefixado` e
`TestWhatsAppAdapter_React_RegistroDeChamadaUsaOCallIdDoWhatsApp` (ambos falham sem a correção).

**Não verificado / fora do escopo:** (1) o WhatsApp NÃO enviou o registro de chamada como mensagem (`callLogMesssage`) ao aparelho vinculado, então não há
outro id a descobrir; (2) outros eventos que usam o id da mensagem (edição, revogação, recibo) não passam por essa tradução; para o registro de chamada
não se aplicam, mas não foram testados; (3) o teste ponta a ponta no celular depende do dono.

## Toque de chamada como convite, não como interrupção (8/nov/2026)

**Problema (dono):** o toque era um `Dialog` modal que cobria a tela e obrigava o atendente a aceitar ou recusar na hora, mesmo no meio de outro
atendimento.

**Decisões (dono):** (1) cartão ancorado no rodapé da lista de tickets, como o WhatsApp Web; (2) "Ligar de volta" como ação no registro de chamada
perdida; (3) ordem: cartão, ligar de volta, plataforma do chamador, prioridade de quem toca; (4) **Ignorar silencia o toque e mantém o cartão na tela até
a chamada acabar** (`call.ended`, por qualquer motivo); (5) fora de `/tickets` o cartão flutua no canto inferior esquerdo, com a mesma largura.

**Entregue neste PR (1 de 4):** `IncomingCallCards` substitui `IncomingCallModal`. Cartão com Atender, Recusar e Ignorar, barra e contagem dos 45 s,
conexão, tipo (voz/vídeo), aviso de navegador sem vídeo e aviso "você está em outra chamada" (Atender bloqueado, Recusar e Ignorar livres). Encaixa no
rodapé da lista de tickets (a lista encolhe) ou flutua. Mostra até 2 cartões (os mais antigos) e conta o resto. Som único: para quando todas foram
ignoradas ou o operador entra em chamada. Ignorar NÃO recusa: o chamador segue ouvindo tocar e outro atendente pode atender.

**Limites conhecidos:** (1) os 45 s são medidos no navegador a partir do `call.incoming`; o engine não manda relógio, então a barra pode divergir um
pouco do celular; (2) "Em espera + atender" com outra chamada em curso não existe (o protocolo talvez não permita); (3) o estado "ignorado" vive só no
navegador (recarregar a página reabre o som da oferta que ainda toca); (4) a prioridade de quem toca (servidor) ainda não foi feita.

**Entregue (3 de 4): aparelho do chamador (8/nov/2026).** O whatsmeow já entrega `CallOffer.RemotePlatform/RemoteVersion` (lê `platform` e `version` do
`<offer>`); o engine os descartava. Agora o engine os repassa em `call.incoming` e `call.missed`, o business os grava em `CallLogs`
(`callerPlatform`, `callerVersion`) e o cartão de chamada e o registro de chamada no chat mostram o aparelho (`Android`, `iPhone`, `Web`, `WhatsApp
Business (Android)`...). **Sem o `~`**: ao contrário do selo da mensagem (estimado pelo formato do id), aqui o valor é **declarado pelo WhatsApp**. Valores
vistos no `lt`: `android`, `web` (dispositivo `:84`, versão `2.3000.x`) e `iphone` (não `ios`); o protocolo lista também `smba`, `smbi`, `ipad`,
`darwin`, `uwp`. Só essas plataformas são gravadas e mostradas (o valor vem do aparelho do chamador, então nunca vira texto livre); versão só com dígitos e
pontos. Só em chamada **recebida**.

**Limites:** (1) o aviso de grupo (`offer_notice`) não traz plataforma (`CallOfferNotice` não tem `CallRemoteMeta`), então nesses registros o aparelho fica
vazio; (2) só `android`, `web` e `iphone` foram vistos de verdade, as demais vêm do catálogo do protocolo; (3) chamadas antigas (antes deste PR) ficam sem
aparelho; (4) coluna nova só no modelo GORM (regra do projeto); no `lt` foi criada à mão com o mesmo DDL.


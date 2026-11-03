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

Entregue (PR da biblioteca): tabela `StickerLibrary` (RLS por empresa, índice único por empresa e hash do conteúdo, então salvar de novo não duplica), `GET/POST/DELETE /stickers`, envio por `stickerId` (id da biblioteca, nunca URL), "Salvar figurinha" no menu da mensagem (recebida ou enviada) e seletor de figurinhas no compositor, só WhatsApp.

Fica para depois (Parte 2): **ler as favoritas dos celulares** e uni-las à biblioteca. Exige ligar `EmitAppStateEventsOnFullSync` e forçar a ressincronização de `regular_low`, e confirmar se o download de uma favorita funciona sem o `fileSHA256` (o `StickerAction` não o traz; `imageHash` talvez sirva).

## Backlog

Ordem acordada: A, B, E, F (C encerrado, D entregue).

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
3. **E. Migração `Whatsapps` → `ChannelConnections`.** `models.Whatsapp` em 38 arquivos de produção, `WhatsappID` em 52, tabela `"Whatsapps"` em 21; 63 arquivos de teste e 65 no frontend (`whatsappId`). Fatias verticais, sem coexistência, suíte inteira verde em cada uma. Alto risco de regressão: começa por um desenho das fatias.
4. **F. Voz** (ver `backlog-voz-transcricao-sintese.md`): transcrição, síntese e chamadas automatizadas pelo flowbuilder; exige estudar os dois repositórios MIT e desacoplar `internal/calls` e `internal/voip`. Primeiro PR: estudo de viabilidade.

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

## Backlog

1. **Vigia de entrega (confiabilidade).** O engine só trata o erro de retorno de `SendMessage`. `Failed to encrypt` é só WARN do whatsmeow, então uma mensagem aceita pelo servidor mas sem `<receipt>` do aparelho fica parada em ack 2 sem aviso. Proposta: guardar o envio aguardando ack, com prazo; sem recibo, virar ack 5 e avisar o operador; reenvio automático limitado (risco de duplicar). **Falta decidir** prazo e política (só aviso ou reenvio).
2. **"Aguardando mensagem" no primeiro envio a contato novo.** O log mostra `Failed to encrypt … no signal session`. Causa não provada; testar texto comum a contato novo.
3. **Migração `Whatsapps` → `ChannelConnections`.** ~90 arquivos de produção; eliminar a dívida do campo `source`, em fatias verticais.
4. **Voz** (ver `backlog-voz-transcricao-sintese.md`): transcrição, síntese e chamadas automatizadas pelo flowbuilder; exige desacoplar o código de chamadas.
5. **Menores:** textos fixos que restam no frontend; áudio WAV, vídeo e sticker enviados pelo operador; `ThumbnailDirectPath/SHA256` do PDF só se a prévia falhar; emojis lado a lado com o WhatsApp Web não comparados.
6. **Edição de mensagens enviadas pelo WhatsApp** (ver `backlog-edicao-mensagens.md`): a API existe no whatsmeow (`BuildEdit`, janela de 20 min); falta contrato, engine, backend e tela. A edição do contato hoje é descartada em silêncio.
7. **Datas/horas de entrega e leitura (estilo "Dados da mensagem" do WhatsApp Web).** Hoje só `Messages.ack` é guardado; o engine descarta o horário (`v.Timestamp`) e o remetente (`v.Sender`) do recibo. Desenho: tabela `MessageReceipts` (mensagem, participante, estágio entregue/lida/reproduzida, horário), só no modelo GORM (no `lt`, criar à mão com o DDL idêntico); o engine envia horário e participante no `message.ack`; `GET /message/:id/receipts`; item "Dados da mensagem" no menu da mensagem, textos nos 3 idiomas. **Decisões do usuário:** vale para grupos (lista por participante) e conversa individual; mensagens antigas ficam sem data, sem backfill. Limite: sem recibo de leitura (confirmação desligada no aparelho do contato) só aparece "entregue".
8. **Enviar mídia como visualização única** (painel de arrastar e soltar, como no WhatsApp Web): hoje só existe a recepção (Epic 11). Falta botão de olho por mídia, campo `viewOnce` no envio e `ViewOnce=true` em `buildMediaMessage`. Decidir antes como a mensagem aparece para nós depois de enviada e se vale para áudio. Risco: o aparelho vinculado pode não ter esse comportamento; validar com a conexão 2 antes de entregar.

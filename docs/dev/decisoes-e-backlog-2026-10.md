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

Ordem acordada: A, B, E, F (C encerrado, D entregue).

**Entregue / encerrado**

- **Vigia de entrega e reenvio manual** (#115/#116): ack 0 parado >2 min vira falha (ack 5); recibo 1–4 depois do 5 desfaz; `UseRetryMessageStore` ligado.
- **Edição de mensagens** (#118–#122): receber e enviar, histórico, selo.
- **Visualização única, recepção** (#124, Epic 11): ao responder no celular principal o Watink baixa a mídia, guarda e mostra como mídia comum com selo.
- **Textos fixos** (#127–#131): `src/` inteiro sem texto fixo em português; guarda `noHardcodedText` sem exceções (valor que não é texto de tela leva `// i18n-ignore`); guarda `noModuleLevelTranslation` impede `tr()` fora de função.
- **Menores (antigo item 5):** WAV (conversão para OGG/Opus), vídeo e PDF já saem como o cliente oficial; emojis locais (#105). Só falta sticker pelo painel (junto da B).
- **"Aguardando mensagem" no primeiro envio (antigo item 2): ENCERRADO por falta de evidência** (30/out/2026). O aviso foi visto no celular do Diomedes antes do vigia e do `UseRetryMessageStore`; não reproduzido (texto a contato novo, sessão local apagada e número fora do WhatsApp foram testados e não geram o sintoma). Reabrir se voltar a ocorrer, anotando o horário para cruzar com o log do engine.

**Fila**

1. **A. Datas/horas de entrega e leitura (estilo "Dados da mensagem" do WhatsApp Web).** Hoje só `Messages.ack` é guardado; o engine descarta o horário (`v.Timestamp`) e o remetente (`v.Sender`) do recibo. Desenho: tabela `MessageReceipts` (mensagem, participante, estágio entregue/lida/reproduzida, horário), só no modelo GORM (no `lt`, criar à mão com o DDL idêntico); o engine envia horário e participante no `message.ack`; `GET /message/:id/receipts`; item "Dados da mensagem" no menu da mensagem, textos nos 3 idiomas. **Decisões do usuário:** vale para grupos (lista por participante) e conversa individual; mensagens antigas ficam sem data, sem backfill. Limite: sem recibo de leitura (confirmação desligada no aparelho do contato) só aparece "entregue".
2. **B. Enviar mídia como visualização única** (painel de arrastar e soltar, como no WhatsApp Web): hoje só existe a recepção. Falta botão de olho por mídia, campo `viewOnce` no envio e `ViewOnce=true` em `buildMediaMessage`; aproveitar para expor sticker no painel. Decidir antes como a mensagem aparece para nós depois de enviada e se vale para áudio. Risco: o aparelho vinculado pode não ter esse comportamento; validar com a conexão 2 antes de entregar.
3. **E. Migração `Whatsapps` → `ChannelConnections`.** `models.Whatsapp` em 38 arquivos de produção, `WhatsappID` em 52, tabela `"Whatsapps"` em 21; 63 arquivos de teste e 65 no frontend (`whatsappId`). Fatias verticais, sem coexistência, suíte inteira verde em cada uma. Alto risco de regressão: começa por um desenho das fatias.
4. **F. Voz** (ver `backlog-voz-transcricao-sintese.md`): transcrição, síntese e chamadas automatizadas pelo flowbuilder; exige estudar os dois repositórios MIT e desacoplar `internal/calls` e `internal/voip`. Primeiro PR: estudo de viabilidade.

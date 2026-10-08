# Proposal

## Why

Atendentes do Watink conversam com clientes só por texto e mídia. Quando o cliente liga pelo WhatsApp, o
engine descarta a oferta (nenhum handler de `events.CallOffer`) e o atendente não fica sabendo; para
retornar, precisa pegar o celular da empresa. Isso quebra o atendimento centralizado e a ligação perdida
não deixa rastro no ticket.

A viabilidade mudou: o `whatsmeow` que o engine usa expõe os eventos de chamada e os internos de sinalização
(`DangerousInternals`), e há implementações abertas em Go puro da pilha de voz (WaCalls, MIT) cobrindo
sinalização `<call>`, criptografia da chave, transporte SRTP/SCTP pelos relays do WhatsApp e o codec MLow,
sem cgo e compatíveis com a imagem `distroless` do engine.

## What Changes

- **Receber chamadas de voz 1:1**: o engine detecta a oferta, avisa o business, e os atendentes elegíveis
  veem um toque em **qualquer tela** com **Atender** e **Recusar**. Perdida ou recusada, a chamada fica
  registrada no ticket.
- **Efetuar chamadas de voz 1:1**: botão "Ligar" no cabeçalho do ticket, pela conexão do ticket.
- **Áudio no navegador**: microfone e alto-falante pelo navegador; o PCM de 16 kHz trafega por **WebSocket**
  até o business (mesmo caminho HTTPS do app, sem abrir porta UDP) e do business ao engine por canal interno.
- **Telemetria de qualidade ao vivo** na tela da chamada: indicador de sinal sempre visível e painel com RTT,
  perda, jitter, taxa de bits e nível de áudio, com alerta quando a qualidade cai. Ao fim, um **resumo de
  qualidade** por chamada fica registrado.
- **Gravação em MP3 (configurável por empresa, desligada por padrão)**: desligada, opcional por chamada ou
  automática. Os dois lados são mixados em um MP3 mono, guardado no S3, e tocam no histórico. Só funciona em
  instalações com S3 configurado.
- **Recurso nativo do core, sem plugin e sem Marketplace**: sempre disponível, controlado só por permissões
  `calls:*`, como o envio de mensagens.
- **Quem recebe o toque**: usuários com `calls:receive` **e** alcance sobre a conexão, com botão **"Pausar
  chamadas"** no navegador de cada operador.
- **Sem teto de chamadas simultâneas**: o risco de banimento é de cada empresa. Fica só o limite técnico de
  **uma chamada ativa por conexão**.
- **Frontend completo**: provedor global de chamadas, modal de chamada recebida, tela de chamada ativa com
  painel de qualidade, botão no ticket, mensagem de chamada no histórico, menu **Chamadas** com histórico e
  player, configuração de gravação, rótulos de permissão e traduções `pt`/`en`/`es`.
- **Documentação**: guia do atendente, guia do administrador (permissões e gravação) e um guia de como o
  Marketplace e o menu lateral funcionam.
- **Engine continua adaptador burro**: sinalização, mídia e medição; comandos por RabbitMQ em **fila
  dedicada** (`engine.go.calls`, consumidor próprio) para que "Atender"/"Encerrar" não esperem atrás de
  mensagens de outras empresas; um WebSocket interno só para o áudio. Regra, permissão, registro, gravação e
  S3 ficam no business.
- **Pré-requisito independente**: corrigir o vazamento entre empresas no SSE (`?rooms=` sem validação).
- **Fora desta entrega**: vídeo, espera/transferência, chamadas em grupo, transcrição, TTS e IVR.

**BREAKING (comportamento)**: ao atualizar, todo usuário de **alcance de empresa** (Administrador, Gerente
Geral) passa a poder receber o toque de chamadas, porque o `RequirePermission` libera esse alcance sem
consultar o cargo. Cargos comuns só recebem depois de ganhar `calls:receive`. A gravação nasce **desligada**.
Constará na nota de atualização.

## Capabilities

### New Capabilities
- `whatsapp-voice-calls`: receber e efetuar chamadas de voz 1:1 do WhatsApp pelo navegador do atendente, com
  sinalização, mídia, ciclo de vida, registro no ticket, permissões, telemetria de qualidade, gravação
  opcional em MP3 e limites de segurança.

### Modified Capabilities
<!-- Nenhuma: openspec/specs/ está vazio, não há requisito existente a alterar. -->

## Impact

- **engine-go**: novo `internal/voip/*` (~15 mil linhas portadas, a maior parte o codec MLow), handlers de
  `events.CallOffer/Accept/Terminate/Transport/RelayLatency`, medição de qualidade, **consumidor dedicado
  `engine.go.calls`** e um WebSocket interno de áudio. Nova dependência `pion/webrtc/v4`. Sem encoder de MP3
  e sem API HTTP de controle. O módulo segue em Go 1.25 (verificado: a pilha compila e seus testes passam).
- **business**: serviço de chamadas (elegibilidade, `CallLogs`, assunção atômica), consumo de eventos
  `call.*`, rotas REST e WebSocket de áudio (nova dependência de WebSocket, hoje inexistente), gravação
  (mixagem e MP3 em Go puro, upload ao S3), tabelas `CallLogs`/`CallRecordingAccess`, permissões, SSE por
  usuário, mensagem de sistema no ticket e correção do SSE.
- **frontend**: ver "Frontend completo" acima; novas dependências: nenhuma (`AudioWorklet` é nativo).
- **infra**: o engine precisa de **saída UDP livre** até os relays do WhatsApp; nenhuma porta de entrada nova.
  Fila RabbitMQ nova `engine.go.calls` (declarada pelo próprio engine).
- **Risco de proxy (ADR 0021)**: a mídia sai por UDP direto e **não** passa pelo proxy da conexão, o que viola
  "fail-closed, nunca cai no IP do servidor". Chamadas ficam **bloqueadas** em conexões com proxy até haver
  transporte de mídia que o respeite.
- **Risco de banimento (ADR 0016)**: chamadas usam um caminho de protocolo mais detectável que texto. É risco
  de cada empresa; fica **documentado** e exibido como aviso permanente, sem bloquear o uso.
- **Privacidade**: a gravação sem aviso ao contato e sem prazo de retenção é decisão do cliente do produto; o
  sistema registra **quem ligou a gravação**, **quem ouviu** e permite **exclusão manual**, e exige um aceite
  de responsabilidade ao ligá-la.
- **Licenças**: WaCalls e `mlow` são MIT (compatíveis com a AGPL-3.0, mantendo os avisos). O encoder de MP3
  (`shine-mp3`, porte em Go do Shine) é **LGPL-2.0** e roda no business; exige manter o aviso e
  disponibilizar as alterações. O astracalls (AGPL) é só referência de design.

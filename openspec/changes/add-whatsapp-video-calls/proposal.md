# Proposal

## Why

O Watink já recebe e faz chamadas de **voz** 1:1 pelo navegador (mudança `add-whatsapp-voice-calls`). Clientes
também ligam por **vídeo**, e hoje o engine trata essas ofertas como voz: o contato é atendido sem imagem.
Quem precisa mostrar um produto, um documento ou um problema não consegue.

A pesquisa (ADR 0032) mostrou que dá para fazer sem codec de vídeo no servidor: o **navegador** codifica e
decodifica H.264 com WebCodecs e o engine só empacota em RTP e repassa pelos relays do WhatsApp, o mesmo
desenho de adaptador burro da voz.

## What Changes

- **Receber vídeo** do contato (o operador vê o contato) em chamadas atendidas.
- **Enviar câmera** do operador ao contato.
- **Iniciar chamada já como vídeo** (botão de videochamada ao lado do de voz no ticket).
- **Upgrade e downgrade no meio da chamada**: de voz para vídeo e de volta, pedido e aceito por qualquer lado.
- **Gravação**: o áudio segue gravado em MP3 como hoje (gatilho `call.state active`); o **vídeo H.264 bruto
  (Annex-B) é guardado à parte** no S3, só para auditoria, sem muxar. O arquivo `.h264` não toca direto no
  navegador. Respeita os mesmos modos (off / opcional / automática) e o mesmo aceite da gravação de voz.
- **Mesmas permissões** da voz: `calls:receive` para atender, `calls:place` para ligar. Nenhuma permissão nova.
- **Só Chromium** (Chrome, Edge, Brave) para vídeo: depende de `MediaStreamTrackProcessor`. Firefox e Safari
  continuam com voz.
- **Engine continua adaptador burro**: sinalização `<video>`, RTP H.264, SRTP e feedback; regra, registro e
  gravação ficam no business.

### Fora do escopo

- Vídeo em grupo, links de chamada, compartilhamento de tela, reações. A `meowcaller` os marca como
  experimentais.
- Codec de vídeo no servidor, transcodificação, muxer MP4.
- Chamada de vídeo por conexão **com proxy**: segue bloqueada (a mídia sai por UDP direto do host; ADR 0021).
- Firefox e Safari.

## Riscos (leia antes de aprovar)

A maior parte deste escopo **nunca foi validada ao vivo por nenhum projeto de referência**:

| Parte | Estado nos projetos de referência |
|---|---|
| Receber vídeo 1:1 | Confirmado ao vivo (meowcaller) |
| **Enviar** vídeo 1:1 | **NOT VALIDATED** (só testes sintéticos) |
| **Upgrade/downgrade no meio** | **NOT VALIDATED** (validado só contra outra implementação) |
| Iniciar chamada já com vídeo, 1:1 | Sem registro de teste com telefone real |

Por isso o plano é em **fases com teste real entre elas**: não se começa a fase seguinte sem a anterior ter
passado com dois números reais. Se uma fase falhar, o escopo seguinte para até a causa ser entendida.
O risco de bloqueio do número (ADR 0016) aumenta com vídeo, e continua sendo da empresa.

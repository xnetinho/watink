# Roteiro de teste manual: chamadas de voz com dois números reais

**Objetivo.** Validar o que **nenhum teste automatizado consegue provar sem o WhatsApp de verdade**:
a sinalização de chamada, o relay UDP, o codec MLow ponta a ponta, a qualidade percebida do áudio e
a medição de RTT. Tudo o mais já tem teste automatizado (ver "O que já está provado" no fim).

> **Estado em 2026-10-06:** nenhuma chamada real foi feita. Este roteiro **não foi executado**.
> Cada passo abaixo tem um campo **Resultado** em branco para você preencher.

## 0. Pré-requisitos

| Item | Como conferir |
|---|---|
| Imagens `:test` do engine e do business publicadas e **o serviço redeployado à força** | `docker service ps` mostra a imagem nova e `CREATED` recente |
| `ENGINE_HOST=<nome-do-engine>` no business (de onde saem `/health`, grupos e áudio) | `docker exec <business> printenv ENGINE_HOST` |
| Saída **UDP** liberada do host do engine | do host: `nc -u -z -v <ip-de-um-relay> 3478` não é conclusivo; o teste real é o passo 3 |
| **Número A** = a conexão do Watink (WhatsApp A conectado no sistema) | menu Conexões mostra **Conectado** |
| **Número B** = um celular comum, que vai ligar e receber | WhatsApp B instalado e com o contato A salvo |
| Conexão A **sem proxy** | Conexões → editar → Proxy = nenhum |
| Usuário **U1** com `calls:receive`, `calls:place`, `calls:read` e fila ligada à conexão A | Acessos → Cargos |
| Usuário **U2** com **só** `calls:read` | idem |
| Usuário **U3** de outra empresa (se houver) | |
| Fone de ouvido no computador (evita eco) e microfone liberado no navegador | |
| S3 configurado (para os passos de gravação) | Configurações → Chamadas **não** mostra o aviso de armazenamento |

Registre: versão/commit das imagens: `__________`  data: `__________`  testador: `__________`

## 1. Receber e atender (cenários: toque, atender, áudio nos dois sentidos, encerrar)

| # | Ação | Critério de aceite | Resultado |
|---|---|---|---|
| 1.1 | U1 logado, em **qualquer tela que não seja Tickets**. B liga para A. | O toque aparece **sobre a tela atual** em até **3 s**, com o nome do contato B, a conexão A e som em loop. O celular A (se houver) também toca. | |
| 1.2 | U1 clica **Atender**. | A tela mostra "Conectando…" e passa a **"Em chamada"** com cronômetro, em até **~10 s**. O cronômetro parte da conexão real. | |
| 1.3 | U1 fala; B escuta. B fala; U1 escuta. | **Áudio nos dois sentidos**, inteligível, sem eco, sem corte contínuo. Anote a latência percebida: `____` ms. | |
| 1.4 | Durante a chamada, observe o indicador de sinal e abra o painel de detalhes. | Indicador visível. Painel mostra **latência, perda, jitter, taxa e níveis**, atualizando a cada ~1 s. O índice aparece como **(estimado)**. | |
| 1.5 | Anote RTT, perda e jitter mostrados: `____ ms / ____ % / ____ ms`. | Valores plausíveis (RTT de dezenas de ms; perda ~0 %). **Se o RTT aparecer "não medido", registre**: o `c2r_rtt` não veio na oferta. | |
| 1.6 | U1 clica **Silenciar**. B fala e U1 fala. | B **deixa de ouvir** U1; U1 continua ouvindo B. "Microfone silenciado" visível. Reativar volta o áudio. | |
| 1.7 | U1 clica **Encerrar**. | B é desconectado em até **3 s**. A tela mostra "Chamada encerrada". | |
| 1.7a | Releia a tela da chamada que acabou de usar. | Há, **em todas as fases** (Chamando/Conectando/Em chamada/Encerrada), o aviso de que chamadas por canal não oficial **aumentam o risco de bloqueio do número** e que esse risco é da empresa. | |
| 1.8 | Abra o ticket de B. | Há uma mensagem **"Chamada de voz recebida"**, com **quem atendeu**, **duração** coerente (±2 s) e situação "Atendida". | |

## 2. Recusar, perder e desistir

| # | Ação | Critério de aceite | Resultado |
|---|---|---|---|
| 2.1 | B liga; U1 clica **Recusar**. | B recebe a recusa. O toque some. Registro **Recusada**. **Confirme no celular A (se tiver)**: o toque também parou (efeito de `reject` na conta inteira, esperado nesta ação explícita). | |
| 2.2 | B liga; **ninguém atende** por 45 s. | O toque some sozinho. Registro **Perdida** (motivo "Não atendida"). Ticket pendente/aberto com a mensagem. | |
| 2.3 | B liga e **desliga antes** de U1 atender. | O toque some para U1. Registro **Perdida**. | |
| 2.4 | **Todos os operadores offline** (feche as abas). B liga. | **O celular A continua tocando normalmente** (o sistema não atendeu nem recusou). Registro **Perdida** ("Sem operador disponível"). Ticket pendente criado. | |

## 3. Atender em outro aparelho

| # | Ação | Critério de aceite | Resultado |
|---|---|---|---|
| 3.1 | B liga; **atenda no celular A** (o aparelho do WhatsApp), não no sistema. | O toque some para U1. Registro **"Atendida em outro aparelho"**. **Nenhum `reject` é enviado** (B não vê recusa). | |

## 4. Fazer uma chamada

| # | Ação | Critério de aceite | Resultado |
|---|---|---|---|
| 4.1 | Em um ticket **individual** de B, U1 clica no botão **ligar**. | Tela "Chamando…". O celular B toca. | |
| 4.2 | B atende. | A tela vira "Em chamada", áudio nos dois sentidos (repita 1.3 a 1.6). | |
| 4.3 | U1 liga de novo e **cancela** antes de B atender. | O toque em B para. Registro **Cancelada**. | |
| 4.4 | U1 liga e B **recusa**. | A chamada termina com o motivo e fica registrada. | |
| 4.5 | U1 liga e B **não atende por 45 s**. | Encerra como não atendida, registrada. | |
| 4.6 | Em um ticket de **grupo**, olhe o cabeçalho. | **Não há botão de ligar.** | |
| 4.7 | **Desconecte** a conexão A e abra um ticket individual. | O botão de ligar fica **desabilitado**, com o motivo ao passar o mouse. | |

## 5. Conexão com proxy (fail-closed)

| # | Ação | Critério de aceite | Resultado |
|---|---|---|---|
| 5.1 | Configure um proxy na conexão A, reconecte. B liga. | **O celular A continua tocando**, o sistema **não** oferece atender. Registro "Conexão com proxy". | |
| 5.2 | No ticket, olhe o botão de ligar. | **Desabilitado**, com a explicação do proxy. | |
| 5.3 | Remova o proxy, reconecte. | Chamadas voltam a funcionar (repita 1.1). | |

## 6. Permissões

| # | Ação | Critério de aceite | Resultado |
|---|---|---|---|
| 6.1 | B liga com **U2** (só `calls:read`) online e **U1 offline**. | U2 **não** vê toque. Registro **Perdida** (sem operador). | |
| 6.2 | U2 abre o menu. | Vê **Chamadas**, **não** vê o botão de ligar nem a seção Chamadas em Configurações. | |
| 6.3 | Usuário **sem nenhuma** `calls:*`. | Não vê o menu Chamadas. | |
| 6.4 | **U1 pausa** as chamadas (botão do telefone no topo). B liga com U1 como único elegível. | O toque **não** aparece para U1; registro **Perdida**. Despause e confirme que volta a tocar. | |
| 6.4a | Com U1 **em chamada**, B2 liga na **mesma conexão**; depois, com U1 ainda em chamada, **outra conexão** recebe uma oferta. | Na mesma conexão: ver 9.3. Em outra conexão: o toque aparece para U1 mas **Atender fica desabilitado** (só dá para Recusar), pois ele já está em uma chamada. | |
| 6.5 | **Dois operadores** elegíveis online (U1 e U4). B liga; ambos clicam Atender quase juntos. | **Só um** vence; o outro vê "já foi atendida". | |

## 7. Falhas de rede e de áudio

| # | Ação | Critério de aceite | Resultado |
|---|---|---|---|
| 7.1 | Em chamada, **derrube a rede do computador do operador** por ~15 s e volte. | Após **10 s** sem o canal de áudio, a chamada é **encerrada**, B é desconectado, registro com falha de conexão. | |
| 7.2 | Negue o microfone ao navegador (ícone do cadeado → Bloquear) e atenda. | A chamada **não conecta**, a tela explica o microfone negado, e a chamada é encerrada como falha. | |
| 7.3 | **Bloqueie a saída UDP** do host do engine (`iptables -A OUTPUT -p udp -j DROP` no host, temporariamente) e atenda uma chamada. | Em **~25 s** a chamada é encerrada com **"Sem áudio: a conexão de mídia não abriu (verifique a saída UDP…)"**. Remova a regra depois. | |
| 7.4 | Em chamada, **reinicie o engine**. | A chamada some do toque; o registro vira **Interrompida**; o sistema volta a funcionar sem intervenção. | |
| 7.5 | Em chamada, ponha B em modo avião por 20 s. | Aparece "O contato parou de enviar áudio" após ~5 s. Anote o que acontece depois: `____` | |

## 8. Gravação (os três modos)

| # | Ação | Critério de aceite | Resultado |
|---|---|---|---|
| 8.1 | Configurações → Chamadas (com `calls:manage`). Escolha **Opcional**. | O **termo de responsabilidade** aparece; **Salvar** só habilita após marcar o aceite. Após salvar, aparece "Aceito por … em …". | |
| 8.2 | Chamada normal no modo **Opcional**; clique **Gravar** no meio; fale 20 s; **Parar**; encerre. | Indicador **"Gravando"** durante a gravação. No histórico há **Ouvir**. | |
| 8.3 | Clique **Ouvir**. | O áudio toca com **as duas vozes**, na ordem em que ocorreram; **duração coerente** (±1 s da gravada); sem distorção, chiado ou voz de robô. | |
| 8.4 | Baixe/abra o MP3 em outro player. | Abre sem erro, mono, 16 kHz. | |
| 8.5 | Modo **Automática**. Faça uma chamada sem clicar em nada. | Começa a gravar quando o cronômetro liga (mídia conectada), nos dois sentidos; recusada ou sem resposta não grava; indicador visível; ao fim há gravação. | |
| 8.6 | Modo **Desligada**. Faça uma chamada. | Nenhum botão de gravar; nenhuma gravação criada. | |
| 8.7 | Com a gravação ouvida, olhe **Auditoria** (consulta ao banco: `CallRecordingAccesses`). | Há uma linha por escuta com **usuário e horário**. | |
| 8.8 | **Excluir** uma gravação (usuário com `calls:delete`). | Pede confirmação; o arquivo some do S3; histórico mostra "Gravação excluída"; existe linha de auditoria `delete`. | |
| 8.9 | Com a gravação ligada, **derrube o S3** e encerre uma chamada. | A chamada é registrada; histórico mostra **"A gravação falhou"**; nenhum áudio parcial disponível. | |

## 9. Carga e isolamento

| # | Ação | Critério de aceite | Resultado |
|---|---|---|---|
| 9.1 | Dispare **muitas mensagens** (campanha ou envio em massa em outra empresa) e, durante o envio, **atenda e encerre** uma chamada. | "Atender" e "Encerrar" respondem **sem esperar** a fila de mensagens (em até ~3 s). | |
| 9.2 | Duas chamadas simultâneas em **duas conexões diferentes**. | Ambas funcionam ao mesmo tempo. Anote o uso de CPU do engine: `____ %` (esperado ≈16 % de um núcleo por chamada). | |
| 9.3 | Uma segunda chamada chega na **mesma conexão** durante uma ativa. | O sistema **não atende nem recusa**; o celular A mostra a chamada em espera; registro **Perdida** ("Ocupado"). | |
| 9.4 | Com **duas empresas**, faça uma chamada na empresa 1. | O toque e a telemetria **só** aparecem para a empresa 1. | |
| 9.5 | Identificador oculto: ligue de um contato cujo WhatsApp entrega **LID** (sem o número). | A chamada cai no contato **existente** da agenda, sem criar um segundo; se não houver telefone conhecido, cria o contato pelo identificador. | |

## 10. Perguntas abertas que só a chamada real responde

Registre a resposta de cada uma. Elas decidem se o desenho precisa de ajuste.

1. A qualidade do áudio é **boa o bastante** para atendimento (MLow a 6 kbps)? Nota 1 a 5: `____`
2. O **RTT** apareceu (vem do `c2r_rtt` da oferta)? Se "não medido", o relay não enviou a medida.
3. O **par `ping`→`pong`** do relay seria confiável para medir RTT? (Hoje **não é usado**; só anotar.)
4. A chamada **atendida pelo sistema** derruba o toque no celular A (esperado) e **não** no de B?
5. Após **N chamadas**, o número A sofreu algum **aviso, restrição ou desconexão** do WhatsApp? `____`
6. O comportamento é igual com a conexão atrás de **NAT estrito** / rede corporativa?

## O que já está provado (não precisa repetir à mão)

Cobertura automatizada dos **74 cenários** do spec: o roteiro acima cobre só o que exige WhatsApp.
Já provado por teste, contra Postgres, RabbitMQ e WebSockets reais, e cada garantia desfeita por
mutação para ver o teste falhar: elegibilidade e paridade com a visibilidade de tickets,
atribuição atômica, idempotência, isolamento entre empresas, permissões, proxy fail-closed,
filas dedicadas (um comando de chamada não espera o consumidor de mensagens), áudio íntegro nos
dois sentidos, descarte com consumidor lento, gravação MP3 decodificada por decodificador
independente, aceite de responsabilidade, auditoria, exclusão, timeouts (3 s, 45 s, 25 s, 10 s),
telemetria só ao operador, fluxo completo com RabbitMQ real (`e2e_flow_test.go`).

## Critério final

- **Aprovado** se todos os passos têm resultado, os critérios de aceite passam, e as respostas
  da seção 10 não exigem mudança de desenho.
- Falha em qualquer passo **bloqueia o merge**. Registre o que falhou e o log relevante
  (`docker service logs` do engine e do business, **sem** `proxyUrl`).

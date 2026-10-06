# Usando as chamadas de voz

Guia do **operador**: como receber e fazer chamadas de voz do WhatsApp direto pelo navegador,
acompanhar a qualidade e o que o sistema ainda **não** faz.

> Se você não vê o toque, o botão de ligar ou o menu **Chamadas**, falta uma permissão no seu
> cargo. Peça ao administrador (veja [Habilitar chamadas](./ENABLING_CALLS.md)).

## O que você precisa

- Uma permissão no seu cargo: **atender e recusar** (`calls:receive`), **ligar** (`calls:place`)
  e/ou **ver o histórico** (`calls:read`).
- **Microfone** no computador e a permissão do navegador para usá-lo.
- A conexão do WhatsApp **conectada**.

## Receber uma chamada

Quando alguém liga para uma conexão sua, aparece um aviso **sobre a tela em que você estiver**
(não precisa estar em Tickets) com o **nome do contato**, a **conexão** que recebeu a ligação e
um som de chamada.

- **Atender**: abre a tela da chamada e liga o áudio.
- **Recusar**: o contato recebe a recusa e o toque some para todos.
- Se **outra pessoa atender primeiro**, o toque some para você e aparece o aviso de que a
  chamada já foi atendida.
- Se o **contato desistir** antes de alguém atender, o toque some e a chamada fica registrada
  como **perdida**.
- Se a **mesma conta de WhatsApp atender em outro aparelho** (o celular, por exemplo), o toque some
  para os operadores e a chamada fica registrada como **atendida em outro aparelho**.
- Se ninguém atender em **45 segundos**, a chamada é encerrada como não atendida.
- Se você **já está em uma chamada**, não consegue atender outra (pode recusar).

**Quem recebe o toque:** só quem tem a permissão, está **online** e cuja visão inclui aquela
conexão (a mesma regra de quem enxerga os tickets dela). Quem pausou as chamadas (abaixo)
não toca.

### Quando ninguém pode atender

Se não houver ninguém elegível online, o sistema **não atende nem recusa**: o celular da conta
continua tocando normalmente e a chamada fica registrada como **perdida**, com um ticket
pendente para o contato.

## Fazer uma chamada

No cabeçalho de um ticket de contato **individual** há o botão de **ligar** (telefone).

1. Clique em ligar. A tela mostra **“Chamando…”**.
2. Quando o contato atender, a tela passa a mostrar **“Em chamada”** com o cronômetro.
3. Para cancelar antes de atender, clique em **Encerrar**; o toque do contato para.

O botão fica **desabilitado**, com o motivo ao passar o mouse, quando:

- a **conexão não está conectada**;
- a conexão tem **proxy** configurado (chamadas não funcionam com proxy);
- você **já está em outra chamada**.

Em **grupos, comunidades e canais** o botão não aparece: só há chamada 1:1.
Se o contato recusar, estiver ocupado ou não atender em 45 segundos, a chamada termina com o
motivo correspondente e fica registrada.

## Durante a chamada

- **Silenciar microfone**: o contato deixa de ouvir você, sem encerrar a chamada. O estado
  “Microfone silenciado” fica visível.
- **Encerrar**: desconecta o contato e libera o áudio.
- **Gravar / Parar gravação**: só aparece se a empresa estiver no modo de gravação **opcional**
  (veja abaixo). Quando está gravando, um indicador **“Gravando”** fica sempre visível.

### Se o microfone não funcionar

Se o navegador **negar o acesso ao microfone**, a chamada não conecta: a tela explica o motivo
e a chamada é encerrada como falha. Libere a permissão do microfone para o site (ícone de
cadeado na barra de endereço) e ligue de novo. Mensagens parecidas aparecem se não houver
microfone ou se o navegador não suportar chamadas.

Se a **conexão de áudio do seu navegador cair por mais de 10 segundos**, a chamada é encerrada e
registrada com falha de conexão.

## Qualidade da chamada

Com a chamada ativa há um **indicador de sinal de três níveis** sempre visível: **Boa**,
**Regular** ou **Ruim**. Clique nele para abrir os detalhes, atualizados a cada segundo:

- **Latência até o relay**, **perda recebida** e **jitter**;
- **taxa de envio e de recebimento**;
- **nível do seu microfone** e **nível do contato**;
- **Índice de qualidade (estimado)**.

Quando a latência, a perda ou o jitter passam do limite, o indicador muda de nível e aparece um
**alerta** na tela, sem encerrar a chamada. Se o **contato parar de enviar áudio por mais de 5
segundos**, aparece o aviso “O contato parou de enviar áudio”.

### O que os números significam (e o que não significam)

- O **índice é uma estimativa** calculada a partir das medições de rede. **Não é** a qualidade
  que o contato percebe.
- A **perda mostrada é a do áudio que você recebe**. A perda no sentido contrário (do seu
  microfone até o contato) **não é medida**.
- A latência mostrada soma o pequeno atraso do buffer de áudio do seu navegador.

## Pausar o recebimento

O botão de **telefone** no topo da tela pausa as chamadas **neste navegador**: você deixa de ver
o toque até retomar, e os outros operadores elegíveis continuam tocando. A preferência fica
guardada no navegador e o sistema também é avisado, para não contar você como disponível.

## Histórico no ticket

Cada chamada vira uma **mensagem de sistema no ticket** do contato: se foi recebida ou
realizada, **quem atendeu**, a **duração** e o resultado (atendida, perdida, recusada ou
interrompida). Se houver **gravação**, há o botão **Ouvir gravação**.

## Menu Chamadas

Quem tem a permissão de histórico vê o menu **Chamadas**: uma tabela com direção, contato,
situação, início, duração, qualidade e gravação, com filtros por **situação** e **direção**.
**Ouvir** toca a gravação (a escuta fica registrada) e, se você puder excluir, **Excluir
gravação** pede confirmação e remove o arquivo definitivamente.

## Aviso de risco

O Watink usa o WhatsApp por um canal **não oficial**. Isso vale para mensagens e também para
**chamadas**, e **chamadas por canal não oficial aumentam o risco de bloqueio do número**. Esse
risco é **da empresa**, não do Watink. Por isso a tela da chamada mostra este aviso o tempo todo.
Use chamadas com moderação nos números que a empresa não pode perder.

## Limitações

- Só **voz 1:1**. Chamadas de **vídeo** e **em grupo** não são oferecidas; o sistema não as recusa
  (o celular continua tocando) e registra que o tipo não é suportado.
- **Uma chamada por conexão** e uma por operador. Se a conexão já está em chamada e chega outra,
  a nova **não é atendida** pelo sistema e fica registrada como perdida por ocupação.
- Conexões com **proxy** não fazem nem recebem chamadas pelo sistema.
- O sistema precisa de **saída de rede UDP** a partir do servidor (veja o guia do administrador).
- **Não há aviso sonoro ao contato** quando a chamada é gravada: isso é responsabilidade da empresa.

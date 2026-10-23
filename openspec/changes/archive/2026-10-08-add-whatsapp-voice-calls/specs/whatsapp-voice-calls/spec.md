# Spec Delta

## Purpose

Permite que atendentes recebam e efetuem chamadas de voz 1:1 do WhatsApp pelo navegador, a partir da conexão
de um ticket, com acompanhamento da qualidade ao vivo, registro no histórico da conversa, gravação opcional
em MP3 e controle de acesso por cargo.

## ADDED Requirements

### Requirement: Disponibilidade como recurso nativo
O sistema SHALL oferecer chamadas de voz como recurso nativo do produto, sem ativação por Marketplace, e SHALL
controlar o acesso exclusivamente por permissões, de modo que nenhuma empresa precise ativar nada para usá-lo.

#### Scenario: Usuário com permissão usa sem ativar nada
- **WHEN** um usuário com a permissão adequada abre o sistema em uma empresa que nunca configurou chamadas
- **THEN** o recurso está disponível para ele, sem etapa de ativação

#### Scenario: Aviso de risco permanente
- **WHEN** um atendente vê a tela de chamada ou o administrador abre o guia de chamadas
- **THEN** o sistema informa que chamadas por canal não oficial aumentam o risco de bloqueio do número e que
  esse risco é da empresa

### Requirement: Permissões por cargo
O sistema SHALL controlar o acesso por permissões independentes no catálogo `recurso:ação`: `calls:receive`
(atender e recusar), `calls:place` (efetuar), `calls:read` (consultar histórico e ouvir gravações),
`calls:delete` (excluir gravações) e `calls:manage` (configurar gravação), e SHALL NOT anexar nenhuma delas a
cargos existentes.

#### Scenario: Cargo comum sem permissão não vê o toque
- **WHEN** chega uma chamada e o usuário tem cargo comum sem `calls:receive`
- **THEN** o toque e os botões de atender/recusar não são exibidos para esse usuário

#### Scenario: Efetuar sem permissão é negado
- **WHEN** um usuário sem `calls:place` tenta iniciar uma chamada
- **THEN** o sistema responde com negação de permissão e nenhuma chamada é iniciada

#### Scenario: Cargos existentes não ganham acesso sozinhos
- **WHEN** o sistema é atualizado em uma empresa com cargos já criados
- **THEN** nenhum cargo comum recebe as permissões de chamada até que um administrador as conceda

#### Scenario: Alcance de empresa já passa
- **WHEN** um usuário de alcance de empresa (Administrador ou Gerente Geral) entra no sistema após a atualização
- **THEN** ele passa a poder receber, efetuar e consultar chamadas, e isso consta na nota de atualização

#### Scenario: Permissões aparecem na tela de Cargos
- **WHEN** um administrador abre a edição de um cargo
- **THEN** as permissões de chamada aparecem na matriz de permissões com descrições legíveis em português

### Requirement: Quem recebe o toque
O sistema SHALL exibir o toque de uma chamada recebida apenas para usuários conectados que possuam
`calls:receive` e alcance sobre a conexão que recebeu a chamada, e SHALL permitir que cada operador pause o
recebimento no próprio navegador.

#### Scenario: Toque restrito por permissão e alcance
- **WHEN** chega uma chamada em uma conexão
- **THEN** só veem o toque os usuários online com `calls:receive` cujo alcance inclui aquela conexão

#### Scenario: Operador pausa o recebimento
- **WHEN** um operador ativa "Pausar chamadas"
- **THEN** ele deixa de ver o toque nesse navegador até reativar, e os demais operadores elegíveis continuam
  vendo

#### Scenario: Ninguém elegível online
- **WHEN** chega uma chamada e nenhum usuário elegível está online
- **THEN** o sistema não atende nem recusa a chamada, o celular da conta continua tocando normalmente e a
  chamada é registrada como perdida

### Requirement: Receber chamada de voz
O sistema SHALL notificar em tempo real os operadores elegíveis quando uma chamada de voz 1:1 chegar a uma
conexão e SHALL permitir que um deles atenda ou recuse, de qualquer tela do sistema.

#### Scenario: Toque em qualquer tela
- **WHEN** chega uma chamada e o operador está em uma tela que não é a de tickets
- **THEN** o toque aparece sobre a tela atual com o nome do contato e a conexão de destino

#### Scenario: Primeiro a atender assume a chamada
- **WHEN** dois operadores tentam atender a mesma chamada quase ao mesmo tempo
- **THEN** apenas o primeiro atendimento é efetivado, o segundo é informado de que a chamada já foi atendida e
  o toque some para os demais

#### Scenario: Recusa encerra o toque
- **WHEN** um operador recusa a chamada
- **THEN** o chamador recebe a recusa, o toque some para todos e o registro marca a chamada como recusada

#### Scenario: Chamada atendida em outro aparelho
- **WHEN** a mesma conta de WhatsApp atende a chamada em outro aparelho vinculado
- **THEN** o toque some para os operadores e o registro marca a chamada como atendida em outro aparelho

#### Scenario: Chamador desiste antes do atendimento
- **WHEN** o chamador cancela antes de alguém atender
- **THEN** o toque some e o registro marca a chamada como perdida

#### Scenario: Ninguém atende
- **WHEN** a chamada toca por mais de 45 segundos sem atendimento
- **THEN** o sistema a encerra como não atendida e registra como perdida

#### Scenario: Chamada em grupo ou de vídeo não é oferecida
- **WHEN** chega uma chamada de vídeo ou em grupo
- **THEN** o sistema não oferece atendimento, não a recusa pelo protocolo e registra uma entrada informando que
  o tipo não é suportado

### Requirement: Efetuar chamada de voz
O sistema SHALL permitir que um operador com `calls:place` inicie uma chamada de voz 1:1 para o contato de um
ticket, usando a conexão daquele ticket.

#### Scenario: Ligar a partir do ticket
- **WHEN** o operador clica em ligar no cabeçalho de um ticket de contato individual
- **THEN** o sistema inicia a chamada pela conexão do ticket, exibe "Chamando…" e muda para a tela de chamada
  ativa quando o contato atender

#### Scenario: Conexão desconectada
- **WHEN** o operador tenta ligar e a conexão do ticket não está conectada
- **THEN** o sistema não inicia a chamada e informa que a conexão precisa estar conectada

#### Scenario: Ticket de grupo, comunidade ou canal
- **WHEN** o ticket é de grupo, comunidade ou canal
- **THEN** o botão de ligar não é oferecido e uma tentativa direta pela API é rejeitada

#### Scenario: Contato recusa ou não atende
- **WHEN** o contato recusa, está ocupado ou não atende em até 45 segundos
- **THEN** a chamada é encerrada com o motivo correspondente e registrada

#### Scenario: Operador cancela enquanto chama
- **WHEN** o operador encerra a chamada antes de o contato atender
- **THEN** o contato deixa de receber o toque e o registro marca a chamada como cancelada

### Requirement: Áudio bidirecional pelo navegador
O sistema SHALL transportar o áudio do microfone do operador até o contato e o áudio do contato até o
alto-falante do operador durante uma chamada ativa, sem exigir abertura de portas de entrada na infraestrutura
da empresa.

#### Scenario: Permissão de microfone negada
- **WHEN** o navegador do operador nega o acesso ao microfone ao atender ou ligar
- **THEN** a chamada não é conectada, o operador é informado do motivo e a chamada é encerrada como falha

#### Scenario: Áudio flui nos dois sentidos
- **WHEN** a chamada está ativa
- **THEN** o contato ouve o operador e o operador ouve o contato, com o estado "em chamada" visível e um
  cronômetro iniciado no momento da conexão real

#### Scenario: Silenciar microfone
- **WHEN** o operador ativa o silêncio
- **THEN** o contato deixa de ouvir o operador sem encerrar a chamada, e o estado de silêncio é visível

#### Scenario: Operador perde a conexão com o servidor
- **WHEN** o canal de áudio do navegador cai por mais de 10 segundos
- **THEN** a chamada é encerrada, o contato é desconectado e o registro indica falha de conexão

### Requirement: Concorrência
O sistema SHALL permitir no máximo uma chamada ativa por conexão e SHALL NOT impor limite de chamadas
simultâneas por empresa.

#### Scenario: Segunda chamada na mesma conexão
- **WHEN** a conexão já tem uma chamada ativa e chega outra oferta
- **THEN** a nova chamada não é atendida pelo sistema, o celular continua tocando e ela é registrada como
  perdida por ocupação

#### Scenario: Operador já em chamada
- **WHEN** um operador que já está em uma chamada ativa tenta atender ou iniciar outra
- **THEN** o sistema nega e mantém a chamada atual

#### Scenario: Várias chamadas de conexões diferentes
- **WHEN** uma empresa tem chamadas ativas em várias conexões ao mesmo tempo
- **THEN** o sistema as mantém todas, sem impor teto por empresa

### Requirement: Telemetria de qualidade durante a chamada
O sistema SHALL medir e exibir ao operador, durante a chamada e atualizado a cada segundo, a latência até o
relay, a perda e o jitter do áudio recebido, a taxa de bits e o nível de áudio nos dois sentidos, além de um
indicador de sinal e de um índice de qualidade estimado, e SHALL alertar visualmente quando a qualidade
degradar.

#### Scenario: Indicador de sinal sempre visível
- **WHEN** a chamada está ativa
- **THEN** a tela mostra um indicador de sinal de três níveis que reflete a qualidade atual

#### Scenario: Painel de detalhes
- **WHEN** o operador expande o painel de qualidade
- **THEN** vê latência, perda, jitter, taxa de bits e nível de áudio de entrada e saída, atualizados a cada
  segundo

#### Scenario: Alerta de degradação
- **WHEN** a latência, a perda ou o jitter ultrapassam os limites de alerta
- **THEN** o indicador muda de nível e a tela exibe um alerta visual sem encerrar a chamada

#### Scenario: Índice rotulado como estimativa
- **WHEN** o sistema exibe o índice de qualidade
- **THEN** ele é identificado como estimado, calculado a partir das medições de rede, e não como a qualidade
  percebida pelo contato

#### Scenario: Limite de medição declarado
- **WHEN** o operador consulta o significado da perda
- **THEN** a tela informa que a perda medida é a do áudio que o operador recebe e que a perda no envio ao
  contato não é medida

#### Scenario: Contato parou de enviar áudio
- **WHEN** nenhum áudio do contato é recebido por mais de 5 segundos durante a chamada ativa
- **THEN** a tela exibe um aviso de que o contato parou de enviar áudio

#### Scenario: Telemetria não vaza entre empresas
- **WHEN** uma chamada de uma empresa gera medições
- **THEN** elas só são entregues ao operador que assumiu aquela chamada

### Requirement: Resumo de qualidade por chamada
O sistema SHALL guardar, ao fim de cada chamada, apenas um resumo agregado da qualidade (médias e piores
valores de latência, perda e jitter, e o índice estimado), sem série temporal.

#### Scenario: Resumo gravado ao encerrar
- **WHEN** uma chamada atendida termina
- **THEN** o histórico passa a exibir o resumo de qualidade daquela chamada

#### Scenario: Identificar chamadas ruins
- **WHEN** um usuário com `calls:read` consulta o histórico
- **THEN** pode ver quais chamadas tiveram qualidade baixa a partir do resumo

### Requirement: Registro no ticket
O sistema SHALL registrar cada chamada como evento no histórico do ticket do contato, com direção, horário,
duração, resultado e usuário responsável.

#### Scenario: Chamada atendida registrada
- **WHEN** uma chamada atendida termina
- **THEN** o histórico mostra a direção, o operador, o horário de início e a duração real em segundos

#### Scenario: Chamada perdida cria contexto
- **WHEN** uma chamada recebida termina sem atendimento e o contato não tem ticket aberto na conexão
- **THEN** o sistema cria um ticket pendente para o contato com o registro da chamada perdida, para que um
  operador retorne

#### Scenario: Chamada perdida em ticket existente
- **WHEN** uma chamada recebida termina sem atendimento e o contato já tem ticket aberto na conexão
- **THEN** o registro é adicionado a esse ticket

#### Scenario: Histórico respeita o escopo
- **WHEN** um usuário consulta o histórico de chamadas
- **THEN** vê apenas chamadas da própria empresa e, sem alcance de empresa, apenas as de tickets que já pode
  ver

### Requirement: Gravação configurável por empresa
O sistema SHALL permitir que cada empresa escolha entre não gravar (padrão), gravar por decisão do operador em
cada chamada, ou gravar todas as chamadas, e SHALL exigir de quem altera essa escolha o aceite explícito de um
texto de responsabilidade, registrando quem aceitou e quando.

#### Scenario: Padrão é não gravar
- **WHEN** uma empresa nunca configurou a gravação
- **THEN** nenhuma chamada é gravada e nenhum botão de gravar é oferecido

#### Scenario: Aceite de responsabilidade
- **WHEN** um administrador com `calls:manage` muda a gravação de "desligada" para outro modo
- **THEN** o sistema exibe o texto de responsabilidade e só aplica a mudança após a confirmação, registrando
  usuário e horário

#### Scenario: Gravação opcional por chamada
- **WHEN** a empresa está no modo opcional e o operador clica em gravar durante a chamada
- **THEN** a gravação começa dali em diante e a tela exibe claramente que a chamada está sendo gravada

#### Scenario: Gravação automática
- **WHEN** a empresa está no modo automático e uma chamada é atendida
- **THEN** a gravação começa na conexão do áudio e o operador vê o indicador de gravação

#### Scenario: Instalação sem armazenamento de objetos
- **WHEN** a instalação não tem o armazenamento S3 configurado
- **THEN** as opções de gravação não são oferecidas e a tentativa direta pela API é rejeitada com explicação

#### Scenario: Falha ao salvar a gravação
- **WHEN** o envio da gravação ao armazenamento falha ao fim da chamada
- **THEN** a chamada é registrada normalmente sem gravação, o histórico indica que a gravação falhou e nenhum
  áudio parcial fica exposto

### Requirement: Formato e conteúdo da gravação
O sistema SHALL produzir a gravação como um único arquivo MP3 mono com os dois lados da conversa mixados, e
SHALL guardar no banco apenas a chave do objeto, nunca uma URL assinada.

#### Scenario: Os dois lados na mesma gravação
- **WHEN** uma chamada gravada termina
- **THEN** o arquivo contém a voz do operador e a do contato juntas, na ordem em que ocorreram

#### Scenario: Arquivo reproduzível
- **WHEN** o MP3 gerado é aberto por um decodificador independente
- **THEN** ele decodifica sem erro, com a duração e a taxa de amostragem esperadas

#### Scenario: Duração coerente
- **WHEN** a gravação é de uma chamada de duração conhecida
- **THEN** a duração do MP3 corresponde à duração da gravação dentro de uma tolerância de um segundo

### Requirement: Acesso, auditoria e exclusão de gravações
O sistema SHALL permitir ouvir gravações a quem tem `calls:read` dentro do seu alcance, SHALL registrar quem
ouviu cada gravação e quando, SHALL permitir exclusão por quem tem `calls:delete`, e SHALL NOT expirar
gravações automaticamente.

#### Scenario: Ouvir uma gravação
- **WHEN** um usuário com `calls:read` toca uma gravação no histórico
- **THEN** o áudio é entregue por uma URL temporária e o acesso é registrado com usuário e horário

#### Scenario: Acesso fora do alcance
- **WHEN** um usuário tenta ouvir a gravação de uma chamada de um ticket que não pode ver
- **THEN** o sistema nega o acesso e não revela se a gravação existe

#### Scenario: Excluir uma gravação
- **WHEN** um usuário com `calls:delete` exclui uma gravação
- **THEN** o arquivo é removido do armazenamento, o histórico indica que a gravação foi excluída e a exclusão é
  registrada com usuário e horário

#### Scenario: Gravações não expiram
- **WHEN** passa muito tempo desde uma gravação
- **THEN** o sistema não a remove automaticamente

#### Scenario: Isolamento entre empresas
- **WHEN** um usuário de uma empresa tenta acessar a gravação de outra
- **THEN** o acesso é negado

### Requirement: Identidade do contato nas chamadas
O sistema SHALL associar a chamada ao contato existente da empresa, inclusive quando o WhatsApp identifica o
chamador por um identificador oculto em vez do telefone.

#### Scenario: Chamador identificado por identificador oculto
- **WHEN** a oferta chega com um identificador oculto e o telefone correspondente é conhecido
- **THEN** a chamada é associada ao contato da agenda que tem aquele telefone, sem criar um segundo contato

#### Scenario: Telefone desconhecido
- **WHEN** a oferta chega com identificador oculto sem telefone conhecido
- **THEN** a chamada é associada a um contato identificado pelo próprio identificador, como ocorre com mensagens

### Requirement: Conexões com proxy
O sistema SHALL NOT atender nem originar chamadas em conexões que tenham proxy configurado, enquanto a mídia da
chamada não puder ser roteada pelo proxy da conexão, para nunca expor o endereço de saída do servidor.

#### Scenario: Chamada recebida em conexão com proxy
- **WHEN** chega uma oferta em uma conexão com proxy configurado
- **THEN** o sistema não oferece atendimento, não abre mídia e o celular continua tocando, com o motivo registrado

#### Scenario: Ligar a partir de conexão com proxy
- **WHEN** o operador tenta ligar por uma conexão com proxy configurado
- **THEN** o botão aparece desabilitado com a explicação e uma tentativa direta pela API é rejeitada

### Requirement: Isolamento entre empresas e canais internos
O sistema SHALL isolar chamadas, gravações e telemetria entre empresas em todas as consultas, notificações e
canais de áudio, SHALL validar as salas de tempo real que cada usuário pode assinar, e SHALL manter o canal
interno de áudio do engine acessível apenas à rede interna e com autenticação.

#### Scenario: Notificação restrita à empresa
- **WHEN** uma chamada chega a uma conexão da empresa A
- **THEN** nenhum usuário da empresa B recebe o toque ou qualquer dado da chamada

#### Scenario: Assinatura de sala de outra empresa é recusada
- **WHEN** um usuário da empresa A solicita receber os eventos em tempo real da empresa B
- **THEN** o sistema ignora a solicitação e ele não recebe nenhum evento da empresa B

#### Scenario: Toque só para o usuário elegível
- **WHEN** dois usuários da mesma empresa estão online e apenas um é elegível para a chamada
- **THEN** apenas o elegível recebe o toque

#### Scenario: Canal de áudio exige autenticação e vínculo
- **WHEN** um cliente abre o canal de áudio de uma chamada
- **THEN** o sistema valida o token do usuário e que a chamada pertence à empresa e ao operador que a assumiu, e
  rejeita qualquer outro

#### Scenario: Canal interno sem credencial
- **WHEN** o engine inicia sem o token interno de áudio configurado
- **THEN** o canal de áudio não é iniciado e o restante do engine continua funcionando

### Requirement: Comandos de chamada não esperam mensagens
O sistema SHALL processar os comandos de controle de chamada (atender, recusar, encerrar, originar) por um
caminho independente da fila de mensagens, de modo que não aguardem o envio de mensagens de nenhuma empresa.

#### Scenario: Atender sob carga de mensagens
- **WHEN** há milhares de mensagens aguardando envio e um operador atende uma chamada
- **THEN** o comando de atender é processado sem esperar essas mensagens

#### Scenario: Encerrar sob carga de mensagens
- **WHEN** o operador encerra a chamada durante uma campanha de envio de outra empresa
- **THEN** o contato é desconectado sem o atraso do envio das mensagens

### Requirement: Encerramento e limpeza
O sistema SHALL liberar todos os recursos de mídia e encerrar a chamada no WhatsApp em qualquer término,
inclusive em falhas, e SHALL encerrar chamadas órfãs após reinício do engine.

#### Scenario: Operador encerra
- **WHEN** o operador encerra a chamada ativa
- **THEN** o contato é desconectado, os recursos de mídia são liberados e o registro é finalizado com a duração

#### Scenario: Contato encerra
- **WHEN** o contato encerra a chamada
- **THEN** a tela do operador fecha, os recursos são liberados e o registro é finalizado

#### Scenario: Reinício do engine com chamada ativa
- **WHEN** o engine reinicia com chamadas registradas como ativas
- **THEN** essas chamadas são marcadas como interrompidas e o toque, se houver, é removido dos operadores

### Requirement: Interface do operador
O sistema SHALL oferecer ao operador, em português, inglês e espanhol, o toque global, a tela de chamada
ativa, o botão de ligar no ticket, a mensagem de chamada no histórico da conversa, um menu de histórico de
chamadas com reprodução das gravações e a configuração de gravação.

#### Scenario: Menu de histórico visível só com permissão
- **WHEN** um usuário sem `calls:read` abre o sistema
- **THEN** o item de menu "Chamadas" não é exibido

#### Scenario: Mensagem de chamada na conversa
- **WHEN** uma chamada termina e o ticket é aberto
- **THEN** a conversa mostra uma entrada de chamada com direção, duração, resultado e, se houver, o acesso à
  gravação

#### Scenario: Configuração só para quem gerencia
- **WHEN** um usuário sem `calls:manage` abre as configurações
- **THEN** a seção de gravação de chamadas não é exibida

#### Scenario: Idiomas
- **WHEN** o usuário usa o sistema em inglês ou espanhol
- **THEN** todos os textos de chamada aparecem no idioma escolhido

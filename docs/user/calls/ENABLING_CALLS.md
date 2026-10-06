# Habilitar as chamadas de voz

Guia do **administrador**: dar às pessoas certas acesso às chamadas, configurar a gravação e o
que a instalação precisa ter. As chamadas são um recurso **nativo**: não há plugin nem botão
**Ativar**. O acesso é só por **permissão de cargo**.

> **Risco de bloqueio.** O Watink acessa o WhatsApp por um canal **não oficial**, e **chamadas por
> canal não oficial aumentam o risco de bloqueio do número**. Esse risco é **da empresa**.
> Decida com cuidado quem pode ligar e receber, e em quais conexões. O aviso também aparece, de
> forma permanente, na tela de toda chamada.

## 1. Quem já passa sem fazer nada

Usuários de **alcance de empresa** (**Administrador** e **Gerente Geral**) passam por todas as
permissões automaticamente. Depois da atualização, eles **passam a poder receber o toque**,
ligar e consultar chamadas. Se você não quer isso, ajuste o alcance deles.

Nenhum outro cargo ganha acesso sozinho: **Atendente**, **Gestor** e os cargos que você criou
**não** recebem as permissões de chamada na atualização. Você concede de propósito.

## 2. Conceder permissões a um cargo

1. No menu lateral abra **Acessos** (exige a permissão `users:read`) e entre na aba **Cargos**.
2. Clique em **Novo Cargo**, ou na linha de um cargo existente para editar.
3. Na matriz **Permissões**, procure o recurso **calls** e marque o que fizer sentido:

| Permissão | O que libera |
|---|---|
| `calls:receive` | Atender e recusar chamadas (e ver o toque). Também permite encerrar e pausar. |
| `calls:place` | Ligar para o contato de um ticket. Também permite encerrar a chamada que fez. |
| `calls:read` | Ver o menu **Chamadas** e o histórico, e **ouvir gravações**. |
| `calls:delete` | **Excluir gravações**. |
| `calls:manage` | Ver e configurar a **gravação** em **Configurações → Chamadas**. |

4. Clique em **Salvar**.

Quem fica com **atender/recusar** só toca se também **enxergar a conexão**: a conexão está ligada a
uma fila do usuário (a mesma regra de quem vê os tickets dela). Quem tem alcance de empresa vê
todas.

### Combinações comuns

- **Atendente que atende e liga e consulta o histórico**: `calls:receive`, `calls:place`, `calls:read`.
- **Supervisor que ouve gravações e gerencia**: `calls:read`, `calls:delete`, `calls:manage`.
- **Operador só de entrada**: apenas `calls:receive`.

## 3. Gravação

A gravação **nasce desligada** em toda empresa. Quem tem `calls:manage` a configura em
**Configurações → Chamadas** (o item só aparece para quem tem essa permissão).

| Modo | O que acontece |
|---|---|
| **Desligada** (padrão) | Nada é gravado e nenhum botão de gravar aparece. |
| **Opcional** | O operador escolhe gravar em cada chamada (botão **Gravar** na tela da chamada). |
| **Automática** | Toda chamada é gravada a partir do momento em que o áudio conecta. |

### O aceite de responsabilidade

Para sair de **Desligada** para qualquer outro modo, o sistema mostra o **termo de
responsabilidade** e só habilita **Salvar** depois que você marca **“Li e aceito o termo de
responsabilidade”**. O sistema registra **quem aceitou e quando** (aparece abaixo do modo). Trocar
entre **Opcional** e **Automática**, ou voltar para **Desligada**, não pede o aceite de novo;
sair de **Desligada** outra vez pede.

> Gravar conversas telefônicas exige base legal e, em muitos casos, o consentimento de quem fala.
> O sistema **não emite aviso sonoro ao contato**. Informar os interlocutores e cumprir a
> legislação (incluindo a LGPD) é **responsabilidade da sua empresa**.

### Onde fica a gravação

- A gravação é um arquivo **MP3 mono** com os dois lados da conversa, guardado no
  **armazenamento de objetos (S3)** da instalação. No banco fica **só a chave** do arquivo.
- **Ouvir** gera um link temporário (5 minutos) e **registra quem ouviu e quando**.
- **Excluir** apaga o arquivo do armazenamento, marca o histórico como excluído e registra quem excluiu.
- As gravações **não expiram sozinhas**: só saem por exclusão manual.
- Se o envio ao armazenamento falhar no fim da chamada, a chamada é registrada normalmente,
  o histórico indica **“A gravação falhou”** e nenhum áudio parcial fica disponível.

## 4. O que a instalação precisa ter

### Armazenamento de objetos (S3)

**Sem S3, não há gravação.** A seção **Chamadas** avisa que “esta instalação não tem
armazenamento de objetos (S3) configurado”, só **Desligada** fica selecionável e a API recusa
qualquer outro modo. As chamadas em si funcionam normalmente. O S3 é configurado pelas variáveis
`S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY` e `S3_SECRET_KEY` (as mesmas já usadas pela Base de
Conhecimento e pelas Atividades).

### Rede do servidor (UDP)

O áudio da chamada passa pelo servidor do Watink até os servidores do WhatsApp por **UDP**. O
servidor do **engine** precisa ter **saída UDP liberada** para a internet (portas altas). **Não é
preciso abrir nenhuma porta de entrada.**

Se a saída UDP estiver bloqueada (firewall, rede corporativa), o áudio nunca conecta. A chamada
chega a ser atendida, mas, depois de **25 segundos** sem a mídia abrir, o sistema a **encerra** e a
registra como **falha** com o motivo **“Sem áudio: a conexão de mídia não abriu (verifique a saída
UDP do servidor)”**, que aparece no histórico do ticket e na tela de Chamadas. Esse motivo
(`media_timeout`) quase sempre indica saída UDP bloqueada, e não um erro do operador. A correção é
liberar a saída UDP do servidor do engine.

### Conexões com proxy

**Conexões com proxy não fazem nem recebem chamadas.** O áudio sairia pelo IP do servidor, o que
anularia o proxy. Para essas conexões o botão de ligar fica desabilitado e as chamadas recebidas
são ignoradas (o celular continua tocando) e registradas com o motivo.

### Variáveis de ambiente

Estas variáveis ligam o áudio entre o servidor da aplicação e o engine. **Sem elas as chamadas
tocam mas não têm áudio.**

| Variável | Onde | Para quê |
|---|---|---|
| `CALLS_AUDIO_TOKEN` | **engine** e **business** (o mesmo valor) | Autentica o canal interno de áudio. **Obrigatória**: o engine **não sobe o canal** sem ela. Use um valor longo e aleatório. |
| `CALLS_AUDIO_URL` | **business** | Endereço interno do canal de áudio do engine, por exemplo `ws://watink-engine:8085`. |
| `CALLS_AUDIO_PORT` | **engine** | Porta do canal de áudio (padrão `8085`). |
| `CALLS_AUDIO_ORIGINS` | **business** (opcional) | Origens extras aceitas no WebSocket do navegador (lista separada por vírgula). |

O canal do engine é **só interno**: nunca o publique em `ports:`, use apenas `expose:`.

## 5. Conferindo

1. Dê `calls:receive` a um cargo e peça a um usuário desse cargo para entrar na conexão.
2. Ligue para o número da conexão de outro celular.
3. O toque deve aparecer para esse usuário em qualquer tela.

Se o toque não aparece, confira: a pessoa tem `calls:receive`? A conexão está ligada a uma fila
dela? Ela está **online** e **não pausou** as chamadas? A conexão tem **proxy**?

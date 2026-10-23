# Nota de atualização: chamadas de voz

Leia antes de atualizar uma instalação que já tem empresas em uso.

## O que muda ao atualizar

1. **Administradores e Gerentes Gerais passam a poder receber o toque de chamadas.**
   Os usuários de **alcance de empresa** passam por todas as permissões sem consultar o cargo,
   então, assim que a atualização entra, quem tem esse alcance **toca** quando chega uma chamada
   nas conexões que enxerga, e pode ligar e consultar o histórico. Se não quiser isso, reduza o
   alcance desses usuários em **Acessos → Usuários**.
2. **Nenhum outro cargo ganha acesso.** Atendente, Gestor e os cargos criados por você **não**
   recebem as permissões `calls:*`. Conceda de propósito em **Acessos → Cargos**
   (veja [Habilitar chamadas](./ENABLING_CALLS.md)).
3. **A gravação nasce desligada** em todas as empresas. Nada é gravado até um usuário com
   `calls:manage` ligar a gravação em **Configurações → Chamadas** e aceitar o termo de
   responsabilidade. Sem armazenamento S3 a gravação não fica disponível.
4. **O sistema passa a depender de saída UDP** a partir do servidor do engine para o áudio das
   chamadas. Sem ela, as chamadas tocam mas terminam sem áudio após 25 segundos.
5. **O tempo real foi endurecido.** O stream de eventos (`/events`) deixou de aceitar salas de outra
   empresa pedidas pelo navegador. Isso corrige um vazamento entre empresas e não exige ação, mas
   conexões que dependiam de salas fora da lista permitida deixam de recebê-las.

## Variáveis de ambiente novas

| Variável | Onde | Observação |
|---|---|---|
| `ENGINE_HOST` | business | Ex.: `watink-engine`. O business monta sozinho as URLs de `/health`, grupos e áudio a partir dele. `ENGINE_HEALTH_URL`, `GROUPS_API_URL` e `CALLS_AUDIO_URL` **deixaram de existir**: remova-as do compose. |
| `CALLS_AUDIO_PORT` | engine | Padrão `8085`. Só `expose`, nunca `ports`. |
| `CALLS_AUDIO_ORIGINS` | business | Opcional. Origens extras do WebSocket do navegador. |

## Ordem sugerida

1. Atualize o **engine** e o **business** juntos (o engine passa a consumir a fila `engine.go.calls`).
2. Configure as variáveis acima.
3. Confirme a saída UDP do servidor do engine.
4. Conceda as permissões aos cargos que vão atender e ligar.
5. Se quiser gravar, configure o S3 e ligue a gravação com o aceite.

## Como reverter

As chamadas são aditivas: sem conceder permissões e sem as variáveis de áudio, nada muda para os
atendentes. Para desligar de vez, remova `calls:*` dos cargos e reduza o alcance dos usuários de
empresa. As tabelas novas (`CallLogs`, `CallRecordingAccesses`) podem ficar sem uso.

# Backlog: edição de mensagens no WhatsApp

Anotado a pedido do usuário em 29/out/2026. Nada disto foi implementado.

## Objetivo

Permitir que o operador **edite uma mensagem já enviada** pelo Watink no WhatsApp, como o app oficial faz: o contato
vê o texto novo e o selo "Editada". É o segundo sentido da mesma funcionalidade: mostrar também quando o **contato**
edita uma mensagem dele.

## Ponto de partida

- **whatsmeow já tem a API:** `Client.BuildEdit(chat, id, novoConteudo)` monta a mensagem de edição, enviada por
  `SendMessage`. A janela de edição do WhatsApp é `whatsmeow.EditWindow` = **20 minutos** depois do envio; passado
  o prazo o servidor recusa. A tela precisa esconder/desabilitar a ação fora da janela, e o backend precisa recusar.
- **Telegram já tem o caminho de entrada:** `edited_message` chega em `channel_inbound.go` (campo `Edited`), atualiza
  o corpo da mensagem existente e avisa a tela com `appMessage` `action: "update"`. É o desenho a espelhar.
- **Contrato wbot:** não há comando de edição. Seguir o padrão do `message.revoke` (evento) e dos comandos de
  `contracts/wbot/contract.go`, com teste de cobertura bidirecional (`wbot_contract_test.go`).

## O que falta (por camada)

1. **Contrato e engine:** comando `message.edit` (envio, com `BuildEdit`) e evento `message.edited` (recebimento).
   Hoje o engine só trata `ProtocolMessage_REVOKE` em `events_message.go`; **qualquer outro `ProtocolMessage`,
   inclusive a edição do contato, cai no `return` e é descartado em silêncio** (pelo código; não testado com uma
   edição real). Garantir o `MessageSecret`/id original ao montar a edição.
2. **Backend:** rota para editar (só mensagem `fromMe`, texto/legenda, dentro de 20 min, ticket aberto, mesma regra de
   permissão do envio), gravar o texto novo e marcar `edited`/`editedAt` no modelo (sem migração incremental: só no
   modelo GORM, como manda o `CLAUDE.md`). No recebimento, atualizar o corpo da mensagem existente.
3. **Frontend:** ação "Editar" no menu da mensagem, visível só no prazo; o texto novo substitui o antigo e o balão
   mostra "Editada". Textos em `pt/en/es`, nada fixo.
4. **Telegram:** `editMessageText` existe na API do Bot; avaliar se entra junto ou depois (capacidade por canal em
   `Capabilities`, como `Forward`).

## Cuidados

- **Só texto e legenda.** O WhatsApp não edita o arquivo de uma mídia, só a legenda.
- **Não reenviar nem criar mensagem nova:** edição mantém o id da mensagem original (ao contrário do reenvio de
  mensagem que falhou, que cria outra).
- **Citações:** quem citou a mensagem editada deve mostrar o texto novo (a citação lê `quotedMsg` do banco).
- **Auditoria:** guardar o texto anterior? Decidir (o app oficial não mostra o histórico, mas um atendimento
  pode precisar dele).
- **Verificar no celular e no log do engine** (recibo, `<message ... edit="1">`), como nas demais mensagens: só
  banco/API não bastam.

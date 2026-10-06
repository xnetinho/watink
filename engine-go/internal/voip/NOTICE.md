# Código de terceiros em `internal/voip`

Este diretório é um porte da pilha de voz do WhatsApp do projeto **WaCalls**.

| Item | Valor |
|---|---|
| Origem | https://github.com/JotaDev66/WaCalls (`internal/voip`) |
| Commit | `edeb31f0427aba896639db503153b777a405eccf` (2026-06-25, v1.0.0) |
| Licença | MIT, cópia em `LICENSE.WaCalls` |
| `media/mlow` | MIT, autor Rajeh Taher, cópia em `media/mlow/LICENSE` |

## Alterações em relação à origem

- Apenas o caminho de import interno (`wacalls/internal/voip/...` para
  `github.com/alltomatos/watinkdev/engine-go/internal/voip/...`).
- Comportamento original preservado por padrão. Os testes originais foram mantidos.
- Adições do Watink, todas desligadas por padrão: o campo `CallManager.DeferPreaccept`
  (uma condicional em `HandleCallOffer`) e o arquivo `call/callmanager_watink.go`
  (`SendPreaccept` e `AbandonCall`). O engine do Watink nunca envia `reject`, e só
  envia `preaccept` depois que o business confirma operador elegível.

## Atualizar

Reaplique o porte a partir de um commit novo da origem, rode
`go test ./internal/voip/...` e atualize o commit acima. Não edite estes arquivos
para regras do Watink: as regras ficam em `internal/calls`, que usa esta pilha.

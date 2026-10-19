# API HTTP interna de Grupos/Comunidades (`internal/groupsapi/`)

> Produto de T0.1 do plano em `docs/agents/plugin-grupos-comunidades.md`. Contrato fechado
> antes de qualquer código — muda aqui primeiro, código depois.
>
> **Estado atual (2026-10-18):** esta API interna do engine-go é a única implementação ativa.
> O SaaS externo izapia foi descontinuado e removido do código; as capturas/respostas desse serviço
> nas seções históricas abaixo permanecem como registro de pesquisa/compatibilidade, não como
> provider suportado nem dependência de execução.

## Por que existe

`enginego.Provider` (o transporte AMQP fire-and-forget usado por toda conexão `engineType:
enginego`) não tem canal de retorno síncrono. Gestão de grupo é request/response por natureza
("lista os grupos" precisa da lista *agora*). Esta API dá ao engine-go uma segunda porta HTTP,
só para esse domínio, chamando o `*whatsmeow.Client` já vivo em memória
(`WhatsAppService.clients[sessionID]`, ver `internal/whatsapp/service.go:73`).

**Nunca sai da rede docker.** Compose usa `expose:`, nunca `ports:`. Sem TLS público, sem rota
externa — é detalhe de implementação de `enginego.Provider`, não API do Watink.

## Transporte

- Porta: env `GROUPS_API_PORT`, default `8084`.
- Auth: header `X-Internal-Token`, valor = env `GROUPS_API_TOKEN`. **Fail-closed**: se
  `GROUPS_API_TOKEN` não estiver setado no ambiente, o servidor **não sobe** (mesma postura de
  `cryptobox`/`PROXY_ENC_KEY` no `business`) — nunca abre a porta sem auth.
- Sessão identificada por `sessionID` (int, o `Whatsapp.ID` do core) no path.

## Envelope

O envelope de resposta interno é `{ok, data, error{code, message, details?}}`; é contrato entre
o engine-go e o provider `enginego` no business:

```json
// sucesso
{ "ok": true, "data": { ... } }

// erro
{ "ok": false, "error": { "code": "NOT_FOUND", "message": "grupo não encontrado" } }
```

Códigos de erro usados aqui: `INVALID_INPUT` (400) · `NOT_FOUND` (404) · `AUTH_FAILED` (401, token interno
ausente/errado) · `NOT_ADMIN` (403, ação exige admin no grupo) · `RATE_LIMITED` (429) ·
`PROVIDER_ERROR` (502, erro do whatsmeow não classificado) · `SESSION_NOT_CONNECTED` (409).

## DTO canônico do Watink

Este é **nosso** DTO — normalizado a partir do whatsmeow no engine-go e consumido pelo provider
`enginego` no business. O JSON do provider descontinuado, capturado ao vivo no anexo histórico,
foi usado na validação inicial do contrato.

```go
type GroupInfo struct {
    JID              string         `json:"jid"`              // "120363xxxxx@g.us" — chave estável ponta a ponta (invariant 5)
    Subject          string         `json:"subject"`
    Description      string         `json:"description"`
    Owner            string         `json:"owner"`             // JID (ou @lid) do criador
    IsCommunity      bool           `json:"isCommunity"`
    IsSubGroup       bool           `json:"isSubGroup"`
    ParentJID        string         `json:"parentJID,omitempty"` // presente quando IsSubGroup
    Announce         bool           `json:"announce"`          // só admin manda mensagem
    Locked           bool           `json:"locked"`            // só admin edita metadados
    MemberAddMode    string         `json:"memberAddMode"`     // "admin_add" | "all_member_add"
    JoinApprovalMode bool           `json:"joinApprovalMode"`
    PictureURL       string         `json:"pictureURL,omitempty"`
    CreatedAt        int64          `json:"createdAt"`         // unix seconds
    Participants      []Participant `json:"participants"`
}

type Participant struct {
    JID          string `json:"jid"`
    PhoneNumber  string `json:"phoneNumber,omitempty"` // vazio quando só @lid resolvido (privacidade)
    DisplayName  string `json:"displayName,omitempty"`
    IsAdmin      bool   `json:"isAdmin"`
    IsSuperAdmin bool   `json:"isSuperAdmin"`
}

type ParticipantResult struct {
    JID    string `json:"jid"`
    Status string `json:"status"` // "ok" | "error"
    Error  string `json:"error,omitempty"`
}

type JoinRequest struct {
    JID         string `json:"jid"`
    RequestedAt int64  `json:"requestedAt"`
}

type CommunityInfo struct {
    GroupInfo
    LinkedGroups []GroupInfo `json:"linkedGroups"`
}
```

**Nota de campo (`memberAddMode`):** whatsmeow expõe isso como `GroupMemberAddMode` com valores
`admin_add`/`all_member_add` — os dois nomes acima são literais do enum nativo, não inventados.

## Anexo histórico — probe real contra o provider descontinuado

Capturado em 2026-08-04 contra uma sessão conectada real do provider então ativo
(`GET /api/v1/sessions/{sid}/groups`); o OpenAPI não tipava `data` (era `description`-only), então
este shape é mantido como registro histórico de compatibilidade:

```json
[
  {
    "created": 1785463827,
    "description": "",
    "group_id": "120363410762725180@g.us",
    "owner": "61852109279407@lid",
    "participants": [
      { "is_admin": true,  "is_super_admin": true,  "jid": "61852109279407@lid" },
      { "is_admin": false, "is_super_admin": false, "jid": "54439448715448@lid" }
    ],
    "subject": "Watink plugins"
  }
]
```

`GET /groups/{groupId}` devolve o mesmo shape de um único grupo (não um array).

O provider descontinuado não retornava alguns campos do DTO canônico (`announce`, `locked`,
`memberAddMode`, `joinApprovalMode`, `pictureURL`, `isCommunity`, `isSubGroup`, `parentJID` e
`phoneNumber`/`displayName` por participante). Esse gap pertence somente à implementação antiga;
o engine-go atual usa diretamente whatsmeow e não depende dessa forma de resposta.

**Atualização (T1.3, 2026-08-04) — comunidade validada ao vivo, shape original estava
errado.** Criada e removida uma comunidade de teste na mesma sessão do probe original
(`POST .../communities` seguido de `POST .../groups/{id}/leave` para limpar). O shape real é
**completamente diferente** do que a inferência original (`CommunityInfo` = grupo +
`linkedGroups`) assumia:

```json
// POST /sessions/{sid}/communities → resposta
{
  "community_id": "120363428629289471@g.us",  // NÃO "group_id"
  "created": 1785847047,
  "description": "",
  "owner": "54439448715448@lid",
  "participants": [{ "is_admin": true, "is_super_admin": true, "jid": "54439448715448@lid" }],
  "subject": "..."
}

// GET /sessions/{sid}/communities/{communityId} → resposta
{
  "groups": [                                   // NÃO "linkedGroups"
    { "group_id": "120363426723543087@g.us", "is_default_sub_group": true, "subject": "..." }
  ],
  "participant_count": 1,
  "participants": ["54439448715448@lid"]        // array de string, NÃO de objeto
}
```

**Resultado do probe histórico:** `GET /communities/{id}` não devolvia `subject`/`owner`/`description` no nível raiz,
e participantes vinham como JID cru sem `is_admin`/`is_super_admin`. Esse gap pertencia ao provider descontinuado.
O contrato atual do engine-go usa whatsmeow nativo (`CreateGroup` com `IsParent`, `GetSubGroups`), implementado e
coberto por testes em `groups_community.go`.

## Rotas (20 — MVP; exclui `groups/join` e `groups/preview`, ver decisão §1 do plano)

Prefixo: nenhum (mux dedicado, só este domínio). `{sessionID}` sempre no path.

| Método | Path | whatsmeow por trás |
|---|---|---|
| GET | `/sessions/{sessionID}/groups` | `GetJoinedGroups` |
| GET | `/sessions/{sessionID}/groups/{groupJID}` | `GetGroupInfo` |
| POST | `/sessions/{sessionID}/groups` | `CreateGroup` |
| POST | `/sessions/{sessionID}/groups/{groupJID}/participants` | `UpdateGroupParticipants` |
| **PUT** | `/sessions/{sessionID}/groups/{groupJID}` | subject/description/announce/locked/memberAddMode/picture — **um único PUT**, ver nota abaixo |
| GET | `/sessions/{sessionID}/groups/{groupJID}/invite` | `GetGroupInviteLink(false)` |
| POST | `/sessions/{sessionID}/groups/{groupJID}/invite/revoke` | `GetGroupInviteLink(true)` |
| POST | `/sessions/{sessionID}/groups/{groupJID}/leave` | `LeaveGroup` |
| POST | `/sessions/{sessionID}/groups/{groupJID}/join-approval-mode` | `SetGroupJoinApprovalMode` |
| GET | `/sessions/{sessionID}/groups/{groupJID}/join-requests` | `GetGroupRequestParticipants` |
| POST | `/sessions/{sessionID}/groups/{groupJID}/join-requests` | `UpdateGroupRequestParticipants` |
| POST | `/sessions/{sessionID}/communities` | `CreateGroup` com `IsParent: true` |
| GET | `/sessions/{sessionID}/communities/{communityJID}` | `GetSubGroups` + `GetGroupInfo` |
| POST | `/sessions/{sessionID}/communities/{communityJID}/groups/{groupJID}` | `LinkGroup` |
| POST | `/sessions/{sessionID}/communities/{communityJID}/groups/{groupJID}/remove` | `UnlinkGroup` |

> **Divergência do desenho original (registrada aqui, T1.2):** o rascunho inicial deste
> documento (T0.1) listava uma rota POST por campo (`/subject`, `/description`, `/announce`,
> `/locked`, `/member-add-mode`, `/picture`). A implementação real (T1.2, `groups_write.go` em
> `internal/whatsapp/` e `internal/groupsapi/`) consolidou tudo num único
> **`PUT /sessions/{sessionID}/groups/{groupJID}`** com todos os campos alterados em um só payload.

## Campo `tenantId` nos payloads de escrita

Toda rota de **escrita** (POST/PUT, exceto leitura) espera um campo `tenantId` (string) no
corpo JSON. Diferente do resto do contrato REST do `business` (que resolve tenant via JWT), a
API interna do engine-go não tem autenticação por tenant — só a sessão. `tenantId` existe
**apenas** para que `classifyGroupWriteError` (abaixo) consiga rotear o evento `session.risk`
para o tenant certo quando uma ação de grupo dispara um sinal de risco. Rotas de leitura não
levam `tenantId`.

## `pictureURL` no PUT de configurações

O campo `pictureURL` do body do PUT carrega **base64 da imagem**, não uma URL — nome herdado do
campo de leitura equivalente (`GroupInfo.pictureURL`, que na resposta *é* uma URL, resolvida
pelo WhatsApp após o upload).

**Excluídas do MVP** (decisão §1 do plano — risco de ban por entrar em massa em grupos de
terceiros): `GET /groups/preview` (`GetGroupInfoFromLink`), `POST /groups/join`
(`JoinGroupWithLink`).

### Payloads de escrita (exemplos)

```json
// POST /groups
{ "subject": "Nome do grupo", "participants": ["551199999999@s.whatsapp.net"] }

// POST /groups/{groupJID}/participants
{ "action": "add" | "remove" | "promote" | "demote", "participants": ["jid1", "jid2"] }
// resposta: { "ok": true, "data": { "participants": [ParticipantResult, ...] } }

// POST /groups/{groupJID}/subject        { "subject": "novo nome" }
// POST /groups/{groupJID}/description    { "description": "novo texto" }
// POST /groups/{groupJID}/announce       { "announce": true }
// POST /groups/{groupJID}/locked         { "locked": true }
// POST /groups/{groupJID}/member-add-mode { "mode": "admin_add" | "all_member_add" }
// POST /groups/{groupJID}/join-approval-mode { "enabled": true }
// POST /groups/{groupJID}/join-requests  { "action": "approve" | "reject", "participants": ["jid1"] }
// POST /communities                       { "name": "Nome da comunidade" }
```

## Rate limiting (T1.4)

Token bucket **por `sessionID`**, aplicado antes de qualquer chamada whatsmeow de escrita
(criar, participantes, configs, convite, join-requests, leave, comunidade). Rejeição →
`429 {ok:false, error:{code:"RATE_LIMITED", message:"..."}}`. Este é o limitador de **defesa
em profundidade** — o limitador principal fica no `groups_service.go` do plugin no `business`
(correção §2.2 do plano).

**Default: 20 ações de escrita por sessão a cada janela de 1 minuto** (engine-go e business).
Não é derivado de dados reais de produção — WhatsApp não publica um limite oficial para ações
administrativas de grupo, e a instância Watink não tinha volume suficiente até o fechamento desta
issue para calibrar empiricamente. É uma estimativa
conservadora por analogia: fica **abaixo** do que se observa tolerado para envio de mensagens
em contas não-verificadas (dezenas por minuto), partindo da premissa de que ações
administrativas (add/remove/promote em massa) têm um padrão mais "visivelmente automatizado"
para os sistemas de detecção da Meta do que mensagens individuais espaçadas. **Ambos os
limites são configuráveis** (`groupsThrottle.limit`/`.window` no business,
`throttle.limit`/`.window` no engine-go) — ajustar aqui exige mudar os dois lados
manualmente, não há config compartilhada entre os dois processos. Reavaliar quando houver
volume real de uso em produção (sinal: taxa de `session.risk` emitido por ações de grupo vs.
mensagens, comparada por período).

## Erros do whatsmeow → HTTP (implementado em T1.2)

`internal/whatsapp/groups_errors.go` define `classifyGroupWriteError`, chamado por toda
operação de escrita em `groups_write.go` — **diferente** de `risk.go`'s `riskIQCodes`
(401/403/429/463 tratados igualmente como risco para envio de mensagem), grupos distinguem:

- **401/429/463** → `ErrGroupRateLimited` → `429 {code: "RATE_LIMITED"}` **e** dispara
  `reportIfRiskSignal` (evento `session.risk`, mesma função de `risk.go`, reusada).
- **403** → `ErrGroupNotAdmin` → `403 {code: "NOT_ADMIN"}` — **não** é reportado como risco.
  "Não sou admin deste grupo" é um erro de permissão normal (ex.: usuário tentando remover
  alguém sem ser admin), não um sinal de que a conta está sendo banida/throttled. Misturar os
  dois dispararia falso-positivo de risco em toda tentativa de ação sem permissão.
- Qualquer outro erro passa adiante sem reclassificação (`400`/`502` conforme o handler).

## Client interface

`internal/whatsapp/client_iface.go` (`WhatsAppClient`) lista os métodos de grupo usados pelas
rotas de leitura (T1.1). As rotas de escrita (T1.2) chamam o `*whatsmeow.Client` diretamente
via `getConnectedClient` — mesmo padrão dos demais métodos de `WhatsAppService` (`SendText`,
`SendPoll`, ...), que também não passam pela interface `WhatsAppClient` (ela hoje só documenta
o subconjunto necessário para os testes offline dos métodos legados de send/download; grupos
seguem a mesma convenção de "teste o que dá pra testar sem client real": branches de
validação/erro no pacote `whatsapp`, happy-path completo via `Backend` fake em
`internal/groupsapi`). Métodos nativos usados: `GetJoinedGroups`, `GetGroupInfo`, `CreateGroup`,
`UpdateGroupParticipants`,
`SetGroupName`, `SetGroupTopic`, `SetGroupAnnounce`, `SetGroupLocked`,
`SetGroupMemberAddMode`, `SetGroupPhoto`, `GetGroupInviteLink`, `LeaveGroup`,
`SetGroupJoinApprovalMode`, `GetGroupRequestParticipants`, `UpdateGroupRequestParticipants`,
`GetSubGroups`, `LinkGroup`, `UnlinkGroup`) — mesmo padrão incremental usado nos PRs #214/#220
(send/download). É o que torna T1.1-T1.3 testáveis offline com um client fake.

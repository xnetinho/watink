package routes

// permissionExemptions é o registro versionado das rotas autenticadas de ESCRITA que não exigem permissão
// recurso:ação. Toda entrada precisa de uma justificativa; sem ela o teste TestPermissionExemptions_TemJustificativa
// falha. Rota nova de escrita entra gateada com auth.RequirePermission, ou entra aqui com o motivo (e passa por revisão).
//
// Chave: "MÉTODO /caminho" exatamente como o Gin registra (com os :parâmetros).
var permissionExemptions = map[string]string{
	"DELETE /api/v1/auth/logout": "encerra a própria sessão; todo usuário autenticado pode sair",
	"PUT /api/v1/me":             "edita o próprio perfil; users:* é para gerir terceiros e UpdateMe nunca aceita campos de RBAC",
	"PUT /api/v1/me/theme":       "preferência visual do próprio usuário",
}

// permissionRoutePrefixSkip lista prefixos que não são autenticação de usuário e por isso fogem do teste de permissão.
var permissionRoutePrefixSkip = map[string]string{
	"/api/v1/internal/": "autenticadas por token interno (X-Internal-Token), não por JWT de usuário; cobertas pelos testes dos middlewares InternalSaaSOnly",
}

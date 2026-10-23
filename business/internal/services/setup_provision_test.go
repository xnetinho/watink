package services

import (
	"errors"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func provisionSeed(t *testing.T) (*SetupService, *domain.ProvisionPlanSpec) {
	t.Helper()
	db := newSetupTestDB(t)
	seedPermissions(t, db)
	spec := &domain.ProvisionPlanSpec{
		Name:             "Pro",
		UsersLimit:       10,
		ConnectionsLimit: 3,
		QueuesLimit:      5,
		PluginQuota:      5,
		Price:            199.90,
		Active:           true,
	}
	return NewSetupService(db), spec
}

func provisionData(email string) domain.TenantSeedData {
	return domain.TenantSeedData{
		CompanyName: "ACME Ltda",
		FirstName:   "Maria",
		LastName:    "Silva",
		Email:       email,
		Password:    "Str0ng!Passw0rd",
		Document:    "12345678000199",
	}
}

func TestProvisionTenantCreatesTenantWithPlanSnapshot(t *testing.T) {
	svc, spec := provisionSeed(t)

	res, err := svc.ProvisionTenant(provisionData("maria@acme.com"), *spec, "key-1")
	if err != nil {
		t.Fatalf("ProvisionTenant: %v", err)
	}
	if res.TenantID == "" || res.OwnerUserID == 0 {
		t.Fatalf("resultado incompleto: %+v", res)
	}

	// Tenant criado com a provisionKey.
	var tenant models.Tenant
	if err := svc.db.First(&tenant, "id = ?", res.TenantID).Error; err != nil {
		t.Fatalf("carregar tenant: %v", err)
	}
	if tenant.ProvisionKey == nil || *tenant.ProvisionKey != "key-1" {
		t.Fatalf("provisionKey = %v, quer key-1", tenant.ProvisionKey)
	}
	if tenant.OwnerID == nil || *tenant.OwnerID != res.OwnerUserID {
		t.Fatalf("ownerId = %v, quer %d", tenant.OwnerID, res.OwnerUserID)
	}

	// Plano do snapshot foi criado com os limites certos.
	var plan models.Plan
	if err := svc.db.First(&plan, "name = ?", "Pro").Error; err != nil {
		t.Fatalf("carregar plano: %v", err)
	}
	if plan.UsersLimit != 10 || plan.ConnectionsLimit != 3 || plan.QueuesLimit != 5 || plan.PluginQuota != 5 {
		t.Fatalf("limites do plano errados: %+v", plan)
	}

	// Assinatura vinculada ao plano.
	var sub models.TenantSubscription
	if err := svc.db.First(&sub, `"tenantId" = ?`, tenant.ID).Error; err != nil {
		t.Fatalf("carregar assinatura: %v", err)
	}
	if sub.PlanID != plan.ID {
		t.Fatalf("assinatura no plano %d, quer %d", sub.PlanID, plan.ID)
	}

	// Guarda de segurança: um tenant registrado remotamente via Modo SaaS
	// NUNCA pode ganhar alcance de plataforma deste core — só o wizard local
	// (InitializeTenant) cria alcance=plataforma.
	var owner models.User
	if err := svc.db.First(&owner, res.OwnerUserID).Error; err != nil {
		t.Fatalf("carregar owner: %v", err)
	}
	if owner.Alcance != "tenant" {
		t.Fatalf("SEGURANÇA: owner de tenant provisionado via SaaS com alcance=%q, quer tenant", owner.Alcance)
	}
}

func TestProvisionTenantIsIdempotent(t *testing.T) {
	svc, spec := provisionSeed(t)

	res1, err := svc.ProvisionTenant(provisionData("maria@acme.com"), *spec, "key-idem")
	if err != nil {
		t.Fatalf("primeira provisão: %v", err)
	}
	// Retry com a MESMA chave: mesmo tenant, sem duplicar.
	res2, err := svc.ProvisionTenant(provisionData("maria@acme.com"), *spec, "key-idem")
	if err != nil {
		t.Fatalf("retry idempotente: %v", err)
	}
	if res1.TenantID != res2.TenantID {
		t.Fatalf("tenantId divergiu no retry: %s != %s", res1.TenantID, res2.TenantID)
	}

	var count int64
	if err := svc.db.Model(&models.Tenant{}).Count(&count).Error; err != nil {
		t.Fatalf("contar tenants: %v", err)
	}
	if count != 1 {
		t.Fatalf("esperava 1 tenant, achou %d", count)
	}
}

func TestProvisionTenantDuplicateEmailIsPermanent(t *testing.T) {
	svc, spec := provisionSeed(t)

	if _, err := svc.ProvisionTenant(provisionData("dup@acme.com"), *spec, "key-a"); err != nil {
		t.Fatalf("primeira provisão: %v", err)
	}
	// Chave diferente, MESMO e-mail (Users.email é único global) → falha permanente.
	_, err := svc.ProvisionTenant(provisionData("dup@acme.com"), *spec, "key-b")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("erro = %v, quer ErrEmailAlreadyExists", err)
	}
}

func TestPushSubscriptionUpsertsPlanAndSubscription(t *testing.T) {
	svc, spec := provisionSeed(t)

	res, err := svc.ProvisionTenant(provisionData("maria@acme.com"), *spec, "key-1")
	if err != nil {
		t.Fatalf("provisão: %v", err)
	}
	var tenant models.Tenant
	if err := svc.db.First(&tenant, "id = ?", res.TenantID).Error; err != nil {
		t.Fatalf("carregar tenant: %v", err)
	}

	newSpec := domain.ProvisionPlanSpec{Name: "Enterprise", UsersLimit: 0, ConnectionsLimit: 20, QueuesLimit: 0, PluginQuota: 50, Price: 999.90, Active: true}
	if err := svc.PushSubscription(tenant.ID, newSpec, "trialing", nil); err != nil {
		t.Fatalf("PushSubscription: %v", err)
	}

	var sub models.TenantSubscription
	if err := svc.db.Preload("Plan").First(&sub, `"tenantId" = ?`, tenant.ID).Error; err != nil {
		t.Fatalf("carregar assinatura: %v", err)
	}
	if sub.Status != "trialing" {
		t.Fatalf("status = %s, quer trialing", sub.Status)
	}
	if sub.Plan.Name != "Enterprise" || sub.Plan.PluginQuota != 50 {
		t.Fatalf("plano da assinatura errado: %+v", sub.Plan)
	}

	// Uma única assinatura por tenant (upsert, não insert duplicado).
	var subCount int64
	if err := svc.db.Model(&models.TenantSubscription{}).Where(`"tenantId" = ?`, tenant.ID).Count(&subCount).Error; err != nil {
		t.Fatalf("contar assinaturas: %v", err)
	}
	if subCount != 1 {
		t.Fatalf("esperava 1 assinatura, achou %d", subCount)
	}
}

func TestProvisionTwoTenantsInSameSchema(t *testing.T) {
	svc, spec := provisionSeed(t)

	// Provisiona 2 tenants no MESMO schema — cada um semeia uma fila
	// "Atendimento Inicial" (cor "#000000"). Antes o unique GLOBAL
	// uni_Queues_name/uni_Queues_color quebrava o segundo tenant; regressão do fix
	// (name único por-tenant, color não único).
	r1, err := svc.ProvisionTenant(provisionData("a@acme.com"), *spec, "k1")
	if err != nil {
		t.Fatalf("tenant 1: %v", err)
	}
	r2, err := svc.ProvisionTenant(provisionData("b@acme.com"), *spec, "k2")
	if err != nil {
		t.Fatalf("tenant 2 (regressão fila multi-tenant): %v", err)
	}
	if r1.TenantID == r2.TenantID {
		t.Fatalf("tenants deveriam ser distintos: %s", r1.TenantID)
	}

	var queueCount int64
	if err := svc.db.Model(&models.Queue{}).Count(&queueCount).Error; err != nil {
		t.Fatalf("contar filas: %v", err)
	}
	if queueCount != 2 {
		t.Fatalf("esperava 2 filas (uma por tenant), achou %d", queueCount)
	}
}

func TestPushSubscriptionUnknownTenant(t *testing.T) {
	svc, spec := provisionSeed(t)
	err := svc.PushSubscription(uuid.New(), *spec, "active", nil)
	if !errors.Is(err, ErrTenantNotFound) {
		t.Fatalf("erro = %v, quer ErrTenantNotFound", err)
	}
}

// 5.3: numa empresa NOVA, só o Administrador (que recebe o catálogo inteiro por
// desenho) tem calls:*; Atendente, Gestor e Gerente Geral não.
func TestProvisionTenant_CallPermissionsOnlyOnAdministrador(t *testing.T) {
	svc, spec := provisionSeed(t)
	require.NoError(t, svc.db.Create(&[]models.Permission{
		{Resource: "calls", Action: "receive", IsSystem: true},
		{Resource: "calls", Action: "place", IsSystem: true},
		{Resource: "calls", Action: "read", IsSystem: true},
		{Resource: "calls", Action: "delete", IsSystem: true},
		{Resource: "calls", Action: "manage", IsSystem: true},
	}).Error)

	res, err := svc.ProvisionTenant(provisionData("calls@acme.com"), *spec, "key-calls")
	require.NoError(t, err)

	has := map[string]int64{}
	var rows []struct {
		Name string
		N    int64
	}
	require.NoError(t, svc.db.Raw(`
		SELECT c.name AS name, COUNT(*) AS n
		FROM cargo_permissoes cp
		JOIN "Cargos" c ON c.id = cp."cargoId"
		JOIN "Permissions" p ON p.id = cp."permissionId"
		WHERE c."tenantId" = ? AND p.resource = 'calls'
		GROUP BY c.name`, res.TenantID).Scan(&rows).Error)
	for _, r := range rows {
		has[r.Name] = r.N
	}
	assert.Equal(t, int64(5), has["Administrador"], "o Administrador recebe o catálogo inteiro")
	for _, name := range []string{"Atendente", "Gestor", "Gerente Geral"} {
		assert.Zero(t, has[name], "%s não pode ganhar calls:* sozinho", name)
	}
}

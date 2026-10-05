package repository

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupContactTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewTestDB(t)
}

func seedTwoTenantsContacts(t *testing.T, db *gorm.DB) (tenantA, tenantB uuid.UUID, contactA, contactB *models.Contact) {
	t.Helper()
	tenantA = uuid.New()
	tenantB = uuid.New()

	require.NoError(t, db.Create(&TenantTest{ID: tenantA, Name: "Tenant A"}).Error)
	require.NoError(t, db.Create(&TenantTest{ID: tenantB, Name: "Tenant B"}).Error)

	contactA = &models.Contact{
		Name:     "Carlos A",
		Number:   "5599991111",
		Email:    "carlos@tenant-a.com",
		IsGroup:  false,
		TenantID: tenantA,
	}
	contactB = &models.Contact{
		Name:     "Diana B",
		Number:   "5599992222",
		Email:    "diana@tenant-b.com",
		IsGroup:  false,
		TenantID: tenantB,
	}
	require.NoError(t, db.Create(contactA).Error)
	require.NoError(t, db.Create(contactB).Error)
	return
}

func TestGORMContactRepo_FindByID_TenantIsolation(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, tenantB, contactA, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	// Buscar contactA com tenantA → deve encontrar
	found, err := repo.FindByID(ctx, contactA.ID, tenantA)
	assert.NoError(t, err)
	assert.NotNil(t, found, "deveria encontrar o contato do próprio tenant")
	assert.Equal(t, "Carlos A", found.Name)

	// Buscar contactA com tenantB → deve retornar nil (isolamento)
	leaked, err := repo.FindByID(ctx, contactA.ID, tenantB)
	assert.NoError(t, err)
	assert.Nil(t, leaked, "VAZAMENTO DE DADOS: encontrou contato de outro tenant via FindByID")
}

func TestGORMContactRepo_Find_TenantIsolation(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, tenantB, _, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	// Find vazio tenantA → só Carlos
	contactsA, err := repo.Find(ctx, tenantA, "")
	assert.NoError(t, err)
	assert.Len(t, contactsA, 1, "tenantA deveria ter exatamente 1 contato")
	assert.Equal(t, "Carlos A", contactsA[0].Name)

	// Find vazio tenantB → só Diana
	contactsB, err := repo.Find(ctx, tenantB, "")
	assert.NoError(t, err)
	assert.Len(t, contactsB, 1, "tenantB deveria ter exatamente 1 contato")
	assert.Equal(t, "Diana B", contactsB[0].Name)
}

func TestGORMContactRepo_FindByNumber_TenantIsolation(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, tenantB, contactA, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	// Buscar número do tenantA com tenantA → deve encontrar
	found, err := repo.FindByNumber(ctx, tenantA, contactA.Number, false)
	assert.NoError(t, err)
	assert.NotNil(t, found, "deveria encontrar contato pelo número no tenant correto")

	// Buscar número do tenantA com tenantB → nil (isolamento)
	leaked, err := repo.FindByNumber(ctx, tenantB, contactA.Number, false)
	assert.NoError(t, err)
	assert.Nil(t, leaked, "VAZAMENTO DE DADOS: encontrou contato de outro tenant via FindByNumber")
}

func TestGORMContactRepo_Delete_TenantIsolation(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, tenantB, contactA, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	// Tentar deletar contactA usando tenantB → não deve afetar
	err := repo.Delete(ctx, contactA.ID, tenantB)
	assert.NoError(t, err, "delete com tenant errado não deveria causar erro do GORM")

	// contactA ainda deve existir para tenantA
	found, err := repo.FindByID(ctx, contactA.ID, tenantA)
	assert.NoError(t, err)
	assert.NotNil(t, found, "delete com tenant errado não deveria ter removido o contato")

	// Deletar com tenant correto
	err = repo.Delete(ctx, contactA.ID, tenantA)
	assert.NoError(t, err)

	found, err = repo.FindByID(ctx, contactA.ID, tenantA)
	assert.NoError(t, err)
	assert.Nil(t, found, "contato deveria ter sido removido após delete com tenant correto")
}

func TestGORMContactRepo_Update_TenantIsolation(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, tenantB, contactA, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	// Tentar update com tenant errado
	domainContact := &domain.Contact{
		ID:       contactA.ID,
		TenantID: tenantB,
	}
	fields := map[string]interface{}{"name": "Hacked Contact"}
	err := repo.Update(ctx, domainContact, fields)
	assert.NoError(t, err, "update com tenant errado não deve causar erro GORM")

	// contactA não deve ter sido alterado
	found, err := repo.FindByID(ctx, contactA.ID, tenantA)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "Carlos A", found.Name, "update com tenant errado não deveria alterar o nome")

	// Update com tenant correto
	domainContact.TenantID = tenantA
	err = repo.Update(ctx, domainContact, map[string]interface{}{"name": "Carlos Updated"})
	assert.NoError(t, err)

	found, err = repo.FindByID(ctx, contactA.ID, tenantA)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "Carlos Updated", found.Name, "update com tenant correto deveria alterar o nome")
}

func TestGORMContactRepo_FindByLID(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, tenantB, _, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	lid := "lid-abc123"
	c := &models.Contact{
		Name:     "LID Contact",
		Number:   "5599009900",
		TenantID: tenantA,
		Lid:      &lid,
	}
	require.NoError(t, db.Create(c).Error)

	// FindByLID com tenant correto
	found, err := repo.FindByLID(ctx, tenantA, lid, false)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "LID Contact", found.Name)

	// FindByLID com tenant errado → nil (isolamento)
	leaked, err := repo.FindByLID(ctx, tenantB, lid, false)
	require.NoError(t, err)
	assert.Nil(t, leaked, "FindByLID com tenant errado deve retornar nil")

	// FindByLID com LID inexistente → nil
	notFound, err := repo.FindByLID(ctx, tenantA, "nonexistent-lid", false)
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestGORMContactRepo_Create(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, _, _, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	contact := &domain.Contact{
		Name:     "New Contact",
		Number:   "5599887766",
		Email:    "new@tenant-a.com",
		TenantID: tenantA,
	}
	err := repo.Create(ctx, contact)
	require.NoError(t, err)

	found, err := repo.FindByNumber(ctx, tenantA, "5599887766", false)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "New Contact", found.Name)
	assert.Equal(t, tenantA, found.TenantID)
}

func TestGORMContactRepo_FindOrCreate_TenantIsolation(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, _, _, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	// FindOrCreate com novo número → deve criar no tenant correto
	created, err := repo.FindOrCreate(ctx, tenantA, "5599993333", "New Contact", "", false, false, "", "")
	assert.NoError(t, err)
	assert.NotNil(t, created, "FindOrCreate deveria retornar o contato criado")
	assert.Equal(t, "New Contact", created.Name)
	assert.Equal(t, tenantA, created.TenantID)

	// FindOrCreate com mesmo número e tenantA → deve retornar existente
	found, err := repo.FindOrCreate(ctx, tenantA, "5599993333", "Other Name", "", false, false, "", "")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID, "FindOrCreate deveria retornar o mesmo contato")
}

func TestGORMContactRepo_FindOrCreate_RefreshesExpiredProfilePicUrl(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, _, _, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	created, err := repo.FindOrCreate(ctx, tenantA, "5599994444", "Ana", "https://cdn.example.com/old.jpg", false, false, "", "")
	require.NoError(t, err)
	require.Equal(t, "https://cdn.example.com/old.jpg", created.ProfilePicUrl)

	// URL da CDN expirou e uma nova chega -- deve substituir a antiga, não
	// ficar travada porque o campo já estava preenchido.
	updated, err := repo.FindOrCreate(ctx, tenantA, "5599994444", "Ana", "https://cdn.example.com/new.jpg", false, false, "", "")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/new.jpg", updated.ProfilePicUrl)
}

// A contact with ticket/message history must delete cleanly instead of
// bouncing off fk_Contacts_tickets (issue #408) — there is nowhere to migrate
// that history to, so Delete cascades Messages/Tickets (and the sibling
// Deals/Protocols/ConversationEmbeddings FKs) for the deleted contact.
func TestGORMContactRepo_Delete_CascadesTicketsAndMessages(t *testing.T) {
	db := setupContactTestDB(t)
	tenantID, _, contactA, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	wa := models.Whatsapp{Name: "wa-1", TenantID: tenantID}
	require.NoError(t, db.Create(&wa).Error)

	ticket := models.Ticket{ContactID: contactA.ID, WhatsappID: wa.ID, TenantID: tenantID}
	require.NoError(t, db.Create(&ticket).Error)

	msg := models.Message{ID: "msg-cascade-1", Body: "oi", TicketID: ticket.ID, TenantID: tenantID}
	require.NoError(t, db.Create(&msg).Error)

	err := repo.Delete(ctx, contactA.ID, tenantID)
	require.NoError(t, err, "delete não deveria falhar com violação de FK")

	var contactCount, ticketCount, msgCount int64
	db.Model(&models.Contact{}).Where("id = ?", contactA.ID).Count(&contactCount)
	db.Model(&models.Ticket{}).Where("id = ?", ticket.ID).Count(&ticketCount)
	db.Model(&models.Message{}).Where("id = ?", msg.ID).Count(&msgCount)
	assert.Zero(t, contactCount, "contato deveria ter sido removido")
	assert.Zero(t, ticketCount, "ticket vinculado deveria ter sido removido em cascata")
	assert.Zero(t, msgCount, "mensagem vinculada deveria ter sido removida em cascata")
}

func TestGORMContactRepo_FindOrCreate_NeverErasesProfilePicUrlWithEmpty(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, _, _, _ := seedTwoTenantsContacts(t, db)
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	created, err := repo.FindOrCreate(ctx, tenantA, "5599995555", "Bruno", "https://cdn.example.com/pic.jpg", false, false, "", "")
	require.NoError(t, err)
	require.Equal(t, "https://cdn.example.com/pic.jpg", created.ProfilePicUrl)

	// Falha transitória de busca de foto chega como string vazia -- nunca
	// deve apagar a URL já persistida.
	updated, err := repo.FindOrCreate(ctx, tenantA, "5599995555", "Bruno", "", false, false, "", "")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/pic.jpg", updated.ProfilePicUrl)
}

// POST /contacts devolvia "id": 0: Create converte domain->model por valor e
// descartava o ID/timestamps gerados no INSERT. O frontend usa esse id logo em
// seguida (iniciar conversa com o contato recém-criado), então ficava com 0.
func TestGORMContactRepo_Create_FillsGeneratedFields(t *testing.T) {
	db := setupContactTestDB(t)
	tenantID := uuid.New()
	repo := NewGORMContactRepo(db)

	c := &domain.Contact{Name: "Novo", Number: "5511988887777", TenantID: tenantID}
	require.NoError(t, repo.Create(context.Background(), c))

	assert.NotZero(t, c.ID, "o ID gerado pelo banco precisa voltar no objeto de domínio")
	assert.False(t, c.CreatedAt.IsZero(), "createdAt gerado precisa voltar")
	assert.False(t, c.UpdatedAt.IsZero(), "updatedAt gerado precisa voltar")

	got, err := repo.FindByID(context.Background(), c.ID, tenantID)
	require.NoError(t, err)
	require.NotNil(t, got, "o ID devolvido deve apontar para a linha criada")
	assert.Equal(t, "Novo", got.Name)
}

// ── Identidade do contato: telefone x LID (mesma pessoa, dois endereços) ─────

func countContacts(t *testing.T, db *gorm.DB, tenantID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.Contact{}).Where(`"tenantId" = ?`, tenantID).Count(&n).Error)
	return n
}

// Caso real: o atendente cadastrou o contato pelo NÚMERO (agenda) e iniciou a
// conversa; a pessoa respondeu e o WhatsApp entregou o remetente como @lid.
// Sem unificar, nascia um segundo contato (com o LID no campo number) e um
// segundo ticket.
func TestGORMContactRepo_FindOrCreate_LIDReplyReusesAgendaContactByPhone(t *testing.T) {
	db := setupContactTestDB(t)
	tenant := uuid.New()
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	agenda := &domain.Contact{Name: "Fulano da Agenda", Number: "558382341576", TenantID: tenant}
	require.NoError(t, repo.Create(ctx, agenda))

	got, err := repo.FindOrCreate(ctx, tenant, "163423740493865", "Fulano", "", false, true, "163423740493865@lid", "558382341576")
	require.NoError(t, err)

	assert.Equal(t, agenda.ID, got.ID, "deve reaproveitar o contato da agenda, não criar outro")
	require.NotNil(t, got.Lid)
	assert.Equal(t, "163423740493865@lid", *got.Lid, "o LID passa a ficar gravado no contato existente")
	assert.Equal(t, "558382341576", got.Number, "o número da agenda não pode ser trocado pelo LID")
	assert.Equal(t, "Fulano da Agenda", got.Name, "o nome que o atendente deu não é sobrescrito")
	assert.Equal(t, int64(1), countContacts(t, db, tenant))
}

func TestGORMContactRepo_FindOrCreate_NextLIDMessageFindsItByLID(t *testing.T) {
	db := setupContactTestDB(t)
	tenant := uuid.New()
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	agenda := &domain.Contact{Name: "Fulano", Number: "558382341576", TenantID: tenant}
	require.NoError(t, repo.Create(ctx, agenda))
	first, err := repo.FindOrCreate(ctx, tenant, "163423740493865", "", "", false, true, "163423740493865@lid", "558382341576")
	require.NoError(t, err)

	// Próxima mensagem: o telefone pode nem vir mais; o LID já está gravado.
	second, err := repo.FindOrCreate(ctx, tenant, "163423740493865", "", "", false, true, "163423740493865@lid", "")
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, int64(1), countContacts(t, db, tenant))
}

func TestGORMContactRepo_FindOrCreate_LIDWithoutKnownPhoneStillCreatesLIDContact(t *testing.T) {
	db := setupContactTestDB(t)
	tenant := uuid.New()
	repo := NewGORMContactRepo(db)

	got, err := repo.FindOrCreate(context.Background(), tenant, "999888777", "Anônimo", "", false, true, "999888777@lid", "")
	require.NoError(t, err)
	require.NotNil(t, got.Lid)
	assert.Equal(t, "999888777@lid", *got.Lid)
	assert.Equal(t, int64(1), countContacts(t, db, tenant))
}

func TestGORMContactRepo_FindOrCreate_KnownPhoneButNoContactYetCreatesWithRealPhone(t *testing.T) {
	db := setupContactTestDB(t)
	tenant := uuid.New()
	repo := NewGORMContactRepo(db)

	got, err := repo.FindOrCreate(context.Background(), tenant, "163423740493865", "Beltrano", "", false, true, "163423740493865@lid", "558382341576")
	require.NoError(t, err)
	assert.Equal(t, "558382341576", got.Number, "com o telefone conhecido, number guarda o telefone (não o LID)")
	require.NotNil(t, got.Lid)
	assert.Equal(t, "163423740493865@lid", *got.Lid)
}

func TestGORMContactRepo_FindOrCreate_PhoneMatchNeverCrossesTenants(t *testing.T) {
	db := setupContactTestDB(t)
	tenantA, tenantB := uuid.New(), uuid.New()
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &domain.Contact{Name: "Do A", Number: "558382341576", TenantID: tenantA}))
	got, err := repo.FindOrCreate(ctx, tenantB, "163423740493865", "Do B", "", false, true, "163423740493865@lid", "558382341576")
	require.NoError(t, err)

	assert.Equal(t, tenantB, got.TenantID, "o contato do tenant A não pode ser reaproveitado pelo B")
	assert.Equal(t, int64(1), countContacts(t, db, tenantA))
}

func TestGORMContactRepo_FindOrCreate_PhoneMatchedContactAlreadyHasOtherLIDIsNotOverwritten(t *testing.T) {
	db := setupContactTestDB(t)
	tenant := uuid.New()
	repo := NewGORMContactRepo(db)
	ctx := context.Background()

	other := "111111111111@lid"
	require.NoError(t, repo.Create(ctx, &domain.Contact{Name: "X", Number: "558382341576", TenantID: tenant, Lid: &other}))

	got, err := repo.FindOrCreate(ctx, tenant, "222222222222", "Y", "", false, true, "222222222222@lid", "558382341576")
	require.NoError(t, err)
	require.NotNil(t, got.Lid)
	assert.Equal(t, "111111111111@lid", *got.Lid, "não troca silenciosamente o LID de um contato que já tem outro")
}

package repository_test

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/infrastructure/repository"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGORMQuickAnswerRepository_CRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewGORMQuickAnswerRepo(db)
	ctx := context.Background()
	tenantA := uuid.New()
	tenantB := uuid.New()

	qaA := models.QuickAnswer{
		TenantID: tenantA,
		Shortcut: "olá",
		Slug:     "ola",
		Message:  "Olá! Como posso ajudar?",
		Type:     "text",
	}
	require.NoError(t, repo.Create(ctx, &qaA))
	assert.NotZero(t, qaA.ID)

	qaB := models.QuickAnswer{
		TenantID: tenantB,
		Shortcut: "olá",
		Slug:     "ola",
		Message:  "Mensagem empresa B",
		Type:     "text",
	}
	require.NoError(t, repo.Create(ctx, &qaB))

	// FindByID escopado por tenant
	found, err := repo.FindByID(ctx, qaA.ID, tenantA)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, qaA.Message, found.Message)

	// Outro tenant não acessa
	other, err := repo.FindByID(ctx, qaA.ID, tenantB)
	require.NoError(t, err)
	assert.Nil(t, other)

	// Find com busca
	list, err := repo.Find(ctx, tenantA, "como")
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// Update
	qaA.Message = "Mensagem alterada"
	require.NoError(t, repo.Update(ctx, &qaA))

	// Delete
	require.NoError(t, repo.Delete(ctx, qaA.ID, tenantA))
	afterDelete, err := repo.FindByID(ctx, qaA.ID, tenantA)
	require.NoError(t, err)
	assert.Nil(t, afterDelete)
}

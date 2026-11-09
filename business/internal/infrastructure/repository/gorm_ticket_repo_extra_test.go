package repository

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGORMTicketRepo_FindOrCreatePending_Creates(t *testing.T) {
	db := setupTicketTestDB(t)
	tenantA, _, _, _ := seedTwoTenantsTickets(t, db)
	repo := NewGORMTicketRepo(db)
	ctx := context.Background()

	newTicket := &domain.Ticket{
		Status:              "pending",
		ContactID:           777,
		ChannelConnectionID: 10,
		TenantID:            tenantA,
		ConversationKey:     "wa:777",
	}

	result, err := repo.FindOrCreatePending(ctx, newTicket)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.ID, 0, "deveria ter gerado um ID")
	assert.Equal(t, "pending", result.Status)
	assert.Equal(t, tenantA, result.TenantID)
}

func TestGORMTicketRepo_FindOrCreatePending_FindsExisting(t *testing.T) {
	db := setupTicketTestDB(t)
	tenantA, _, _, _ := seedTwoTenantsTickets(t, db)
	repo := NewGORMTicketRepo(db)
	ctx := context.Background()

	// Primeiro chamada — cria
	ticket := &domain.Ticket{
		Status:              "pending",
		ContactID:           888,
		ChannelConnectionID: 20,
		TenantID:            tenantA,
		ConversationKey:     "wa:888",
	}
	first, err := repo.FindOrCreatePending(ctx, ticket)
	require.NoError(t, err)
	require.NotNil(t, first)

	// Segunda chamada com mesmos dados — deve retornar o existente
	second, err := repo.FindOrCreatePending(ctx, ticket)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, first.ID, second.ID, "FindOrCreatePending deveria retornar o ticket existente")
}

func TestGORMTicketRepo_FindByID_NotFound(t *testing.T) {
	db := setupTicketTestDB(t)
	tenantA, _, _, _ := seedTwoTenantsTickets(t, db)
	repo := NewGORMTicketRepo(db)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, 99999, tenantA)
	assert.NoError(t, err)
	assert.Nil(t, found, "ID inexistente deve retornar nil")
}

func TestGORMTicketRepo_FindOpenByConversation_NotFound(t *testing.T) {
	db := setupTicketTestDB(t)
	tenantA, _, _, _ := seedTwoTenantsTickets(t, db)
	repo := NewGORMTicketRepo(db)
	ctx := context.Background()

	found, err := repo.FindOpenByConversation(ctx, tenantA, 99999, "inexistente")
	assert.NoError(t, err)
	assert.Nil(t, found, "conversa inexistente deve retornar nil")
}

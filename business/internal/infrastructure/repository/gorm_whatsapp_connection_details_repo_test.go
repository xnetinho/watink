package repository

import (
	"context"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
)

func TestWhatsAppConnectionDetailsRepo_RoundTrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewGORMWhatsAppConnectionDetailsRepo(db)
	input := &models.WhatsAppConnectionDetails{ConnectionID: 42, ProxyMode: "single", SyncHistory: true, WID: "5511@s.whatsapp.net"}
	if err := repo.Save(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	got, err := repo.FindByConnectionID(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ProxyMode != input.ProxyMode || !got.SyncHistory || got.WID != input.WID {
		t.Fatalf("details = %+v", got)
	}
}

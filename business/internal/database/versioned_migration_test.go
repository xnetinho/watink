package database

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
)

func TestMigrateVersioned_IsIdempotent(t *testing.T) {
	db := testutil.NewTestDB(t)
	if err := MigrateVersioned(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var first int64
	if err := db.Table("SchemaMigrations").Where("version = 1").Count(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := MigrateVersioned(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var second int64
	if err := db.Table("SchemaMigrations").Where("version = 1").Count(&second).Error; err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 1 {
		t.Fatalf("baseline rows = %d then %d, expected one", first, second)
	}
	if !db.Migrator().HasTable(&models.Tenant{}) {
		t.Fatal("baseline did not create the tenant table")
	}
}

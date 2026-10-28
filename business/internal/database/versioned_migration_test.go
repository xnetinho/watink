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

// Um banco que já aplicou o baseline (v1) tem que receber as colunas da v2: o baseline roda uma vez só, então uma coluna
// nova no modelo GORM não chegaria a ele. Simula o banco antigo removendo as colunas e a linha da v2.
func TestMigrateVersioned_AplicaV2EmBancoComBaseline(t *testing.T) {
	db := testutil.NewTestDB(t)
	if err := MigrateVersioned(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	for _, ddl := range []string{
		`ALTER TABLE "Whatsapps" DROP COLUMN IF EXISTS "presencePolicy"`,
		`ALTER TABLE "Whatsapps" DROP COLUMN IF EXISTS "presenceIdleMinutes"`,
		`DELETE FROM "SchemaMigrations" WHERE "version" = 2`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := MigrateVersioned(t.Context(), db); err != nil {
		t.Fatalf("v2 em banco com baseline: %v", err)
	}
	var policy string
	var idle int
	row := db.Raw(`SELECT column_default FROM information_schema.columns WHERE table_name='Whatsapps' AND column_name='presencePolicy'`).Row()
	if err := row.Scan(&policy); err != nil || policy != "'never'::text" {
		t.Fatalf("presencePolicy: default %q err %v (esperado never)", policy, err)
	}
	row = db.Raw(`SELECT column_default::int FROM information_schema.columns WHERE table_name='Whatsapps' AND column_name='presenceIdleMinutes'`).Row()
	if err := row.Scan(&idle); err != nil || idle != 5 {
		t.Fatalf("presenceIdleMinutes: default %d err %v (esperado 5)", idle, err)
	}
	var n int64
	db.Table("SchemaMigrations").Where("version = 2").Count(&n)
	if n != 1 {
		t.Fatalf("v2 registrada %d vezes", n)
	}
	// idempotente
	if err := MigrateVersioned(t.Context(), db); err != nil {
		t.Fatalf("segunda execução: %v", err)
	}
}

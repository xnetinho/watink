package database

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGormLogLevel(t *testing.T) {
	cases := map[string]logger.LogLevel{
		"":        logger.Warn,
		"warn":    logger.Warn,
		"WARN":    logger.Warn,
		"  info ": logger.Info,
		"info":    logger.Info,
		"error":   logger.Error,
		"silent":  logger.Silent,
		"debug":   logger.Warn,
		"banana":  logger.Warn,
	}
	for in, want := range cases {
		if got := gormLogLevel(in); got != want {
			t.Errorf("gormLogLevel(%q) = %v, esperado %v", in, got, want)
		}
	}
}

// loggedDB abre uma conexão no MESMO banco/esquema do testutil, mas com o logger de produção escrevendo num
// buffer, para provar o que o servidor realmente imprime.
func loggedDB(t *testing.T, level string) (*gorm.DB, *bytes.Buffer) {
	t.Helper()
	base := testutil.NewTestDB(t)
	var dsn string
	if err := base.Raw(`SELECT current_setting('search_path')`).Scan(&dsn).Error; err != nil {
		t.Fatal(err)
	}
	schema := strings.TrimSpace(strings.Split(dsn, ",")[0])
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:" + envOr("DB_PASS", "watink_secret_pass") + "@" + envOr("DB_HOST", "localhost") + ":" + envOr("DB_PORT", "5432") + "/" + envOr("DB_NAME", "watink") + "?sslmode=disable"
	}
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	var buf bytes.Buffer
	db, err := gorm.Open(postgres.Open(url+sep+"search_path="+schema), &gorm.Config{Logger: newGormLoggerTo(&buf, level)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db, &buf
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// "Não achei" já é tratado pelo chamador; o logger não pode imprimir isso como erro a cada 30 s.
func TestGormLogger_RecordNotFoundIsSilent(t *testing.T) {
	db, buf := loggedDB(t, "warn")
	var m models.Message
	if err := db.Where("id = ?", "nao-existe").First(&m).Error; err == nil {
		t.Fatal("esperava gorm.ErrRecordNotFound")
	}
	if out := buf.String(); strings.TrimSpace(out) != "" {
		t.Fatalf("a ausência de linha não pode ser impressa: %q", out)
	}
}

// Em nível info as consultas aparecem, mas o valor do parâmetro (texto de mensagem de cliente) não.
func TestGormLogger_DoesNotLeakParameterValues(t *testing.T) {
	db, buf := loggedDB(t, "info")
	secret := "segredo-do-cliente-12345"
	var m models.Message
	_ = db.Where("id = ?", secret).First(&m).Error
	out := buf.String()
	if !strings.Contains(out, `"Messages"`) {
		t.Fatalf("em info a consulta deve aparecer no log: %q", out)
	}
	if strings.Contains(out, secret) {
		t.Fatalf("o valor do parâmetro vazou para o log: %q", out)
	}
}

// Erro de SQL real continua visível no nível padrão.
func TestGormLogger_RealSQLErrorIsStillLogged(t *testing.T) {
	db, buf := loggedDB(t, "warn")
	if err := db.Exec("SELECT * FROM tabela_que_nao_existe").Error; err == nil {
		t.Fatal("esperava erro de SQL")
	}
	if !strings.Contains(buf.String(), "tabela_que_nao_existe") {
		t.Fatalf("erro real de SQL precisa aparecer no log: %q", buf.String())
	}
}

// Consulta lenta continua visível em warn.
func TestGormLogger_SlowQueryIsLogged(t *testing.T) {
	db, buf := loggedDB(t, "warn")
	if err := db.Exec("SELECT pg_sleep(0.35)").Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(buf.String()), "SLOW SQL") {
		t.Fatalf("consulta lenta precisa aparecer: %q", buf.String())
	}
}

// silent não imprime nem erro.
func TestGormLogger_SilentPrintsNothing(t *testing.T) {
	db, buf := loggedDB(t, "silent")
	_ = db.Exec("SELECT * FROM tabela_que_nao_existe").Error
	if strings.TrimSpace(buf.String()) != "" {
		t.Fatalf("silent não deve imprimir nada: %q", buf.String())
	}
}

// O elo que liga a configuração ao servidor: newGormLogger() (o que Connect usa) lê DB_LOG_LEVEL do ambiente.
// Observa-se pelo efeito: em info a consulta aparece, em warn não. A saída padrão é capturada.
func TestNewGormLogger_ReadsDBLogLevelFromEnvironment(t *testing.T) {
	run := func(level string) string {
		t.Setenv("DB_LOG_LEVEL", level)
		db, _ := loggedDB(t, "warn")
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		old := os.Stdout
		os.Stdout = w
		restore := func() { os.Stdout = old }
		defer restore()
		db.Session(&gorm.Session{Logger: newGormLogger()}).Exec("SELECT 1")
		restore()
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if out := run("info"); !strings.Contains(out, "SELECT 1") {
		t.Fatalf("DB_LOG_LEVEL=info deve imprimir a consulta: %q", out)
	}
	if out := run(""); strings.Contains(out, "SELECT 1") {
		t.Fatalf("sem DB_LOG_LEVEL o padrão é warn e não imprime consulta comum: %q", out)
	}
}

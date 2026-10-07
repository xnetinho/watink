package database

import (
	"io"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/gorm/logger"
)

// slowQueryThreshold é o tempo a partir do qual uma consulta é impressa em nível warn.
const slowQueryThreshold = 200 * time.Millisecond

// gormLogLevel converte DB_LOG_LEVEL em nível do GORM. Ausente ou inválido vira warn: um erro de digitação em
// produção não pode ligar o nível info.
func gormLogLevel(raw string) logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}

// newGormLogger é o logger do GORM em produção.
//   - IgnoreRecordNotFoundError: "não achei" é estado normal (instância sem Modo SaaS, ticket sem fluxo) e o
//     chamador já trata gorm.ErrRecordNotFound; imprimir isso enterrava os erros reais.
//   - ParameterizedQueries: o valor dos parâmetros (texto de mensagem de cliente, ids) não vai para o log, nem em info.
func newGormLogger() logger.Interface {
	return newGormLoggerTo(os.Stdout, os.Getenv("DB_LOG_LEVEL"))
}

func newGormLoggerTo(w io.Writer, level string) logger.Interface {
	return logger.New(log.New(w, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             slowQueryThreshold,
		LogLevel:                  gormLogLevel(level),
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
	})
}

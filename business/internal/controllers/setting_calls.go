package controllers

import (
	"strings"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"gorm.io/gorm"
)

// Chaves por empresa da gravação de chamadas (módulo Chamadas).
const (
	CallRecordingModeKey  = "callRecordingMode"
	callRecordingAckByKey = "callRecordingAckBy"
	callRecordingAckAtKey = "callRecordingAckAt"
)

// Modos de gravação. Ausente ou desconhecido equivale a "off": ninguém grava por
// acidente.
const (
	CallRecordingOff      = "off"
	CallRecordingOptional = "optional"
	CallRecordingAuto     = "auto"
)

// NormalizeCallRecordingMode devolve um dos três modos; qualquer outro valor
// (inclusive vazio) vira "off".
func NormalizeCallRecordingMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case CallRecordingOptional:
		return CallRecordingOptional
	case CallRecordingAuto:
		return CallRecordingAuto
	}
	return CallRecordingOff
}

// isCallRecordingKey diz se a chave pertence à gravação de chamadas. Essas
// chaves só mudam pela rota própria (aceite de responsabilidade + calls:manage),
// nunca pelo PUT /settings/:key genérico.
func isCallRecordingKey(key string) bool {
	switch key {
	case CallRecordingModeKey, callRecordingAckByKey, callRecordingAckAtKey:
		return true
	}
	return false
}

// callRecordingModeOf lê o modo de gravação da empresa; ausente = "off".
func callRecordingModeOf(db *gorm.DB, tenantID interface{}) string {
	var s models.Setting
	err := db.Session(&gorm.Session{NewDB: true}).
		Where(`key = ? AND "tenantId" = ?`, CallRecordingModeKey, tenantID).First(&s).Error
	if err != nil {
		return CallRecordingOff
	}
	return NormalizeCallRecordingMode(s.Value)
}

// hideCallRecordingSettings devolve a lista sem as chaves de gravação de chamadas
// (cópia; não altera a original). Para quem lê sem calls:manage, ausente equivale
// a "off", então omitir a chave é fiel ao comportamento real da empresa só quando
// ela está desligada; por isso as chaves saem da lista em vez de mentir um valor.
func hideCallRecordingSettings(in []models.Setting) []models.Setting {
	out := make([]models.Setting, 0, len(in))
	for _, s := range in {
		if !isCallRecordingKey(s.Key) {
			out = append(out, s)
		}
	}
	return out
}

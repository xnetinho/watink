package controllers

import (
	"github.com/alltomatos/watinkdev/business/internal/calls"
	"github.com/alltomatos/watinkdev/business/internal/models"
)

// As chaves e os modos de gravação de chamadas vivem em internal/calls (fonte
// única); aqui só se reaproveitam para o controller de settings.
const (
	CallRecordingModeKey  = calls.SettingRecordingMode
	callRecordingAckByKey = calls.SettingRecordingAckBy
	callRecordingAckAtKey = calls.SettingRecordingAckAt

	CallRecordingOff      = calls.CallRecordingOff
	CallRecordingOptional = calls.CallRecordingOptional
	CallRecordingAuto     = calls.CallRecordingAuto
)

// NormalizeCallRecordingMode devolve um dos três modos; qualquer outro valor
// (inclusive vazio) vira "off".
func NormalizeCallRecordingMode(v string) string { return calls.NormalizeRecordingMode(v) }

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

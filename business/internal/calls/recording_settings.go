package calls

import (
	"errors"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

var (
	ErrAckRequired = errors.New("ligar a gravação exige o aceite do termo de responsabilidade")
	ErrBadMode     = errors.New("modo de gravação inválido (use off, optional ou auto)")
)

// RecordingConfig é o que a tela de configuração lê.
type RecordingConfig struct {
	Mode  string     `json:"mode"`
	AckBy *int       `json:"ackBy"`
	AckAt *time.Time `json:"ackAt"`
	// Available é falso quando a instalação não tem armazenamento de objetos: as
	// opções de gravação não são oferecidas.
	Available bool `json:"available"`
}

// RecordingConfig lê o modo e o último aceite da empresa.
func (s *Service) RecordingConfig(tenantID uuid.UUID) RecordingConfig {
	cfg := RecordingConfig{Mode: s.recordingMode(tenantID), Available: s.rec.Available()}
	var by models.Setting
	if s.fresh().Where(`key = ? AND "tenantId" = ?`, SettingRecordingAckBy, tenantID).First(&by).Error == nil {
		if n, err := strconv.Atoi(by.Value); err == nil {
			cfg.AckBy = &n
		}
	}
	var at models.Setting
	if s.fresh().Where(`key = ? AND "tenantId" = ?`, SettingRecordingAckAt, tenantID).First(&at).Error == nil {
		if t, err := time.Parse(time.RFC3339, at.Value); err == nil {
			cfg.AckAt = &t
		}
	}
	return cfg
}

func (s *Service) putSetting(tenantID uuid.UUID, key, value string) error {
	return s.fresh().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}, {Name: "tenantId"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updatedAt"}),
	}).Create(&models.Setting{Key: key, TenantID: tenantID, Value: value}).Error
}

// SetRecordingMode muda o modo de gravação da empresa. Sair de "off" para outro
// modo EXIGE o aceite do termo (ack=true) e grava quem aceitou e quando; ficar em
// "off" ou voltar para "off" nunca exige. Sem armazenamento, só "off" é aceito.
func (s *Service) SetRecordingMode(tenantID uuid.UUID, userID int, mode string, ack bool) error {
	switch mode {
	case CallRecordingOff, CallRecordingOptional, CallRecordingAuto:
	default:
		return ErrBadMode
	}
	if mode != CallRecordingOff {
		if !s.rec.Available() {
			return ErrRecordingNoS3
		}
		if s.recordingMode(tenantID) == CallRecordingOff && !ack {
			return ErrAckRequired
		}
	}
	if err := s.putSetting(tenantID, SettingRecordingMode, mode); err != nil {
		return err
	}
	if mode != CallRecordingOff && ack {
		if err := s.putSetting(tenantID, SettingRecordingAckBy, strconv.Itoa(userID)); err != nil {
			return err
		}
		return s.putSetting(tenantID, SettingRecordingAckAt, s.now().UTC().Format(time.RFC3339))
	}
	return nil
}

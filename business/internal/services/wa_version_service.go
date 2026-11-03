package services

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var waVersionPattern = regexp.MustCompile(`^2\.\d{1,5}\.\d{1,12}$`)

// ErrInvalidWAVersion indica um número de versão fora do formato 2.x.y (ex.: 2.3000.1052747707).
var ErrInvalidWAVersion = errors.New("wa version: formato inválido, use 2.3000.1052747707")

// ErrInvalidWAVersionMode indica um modo diferente de auto/manual.
var ErrInvalidWAVersionMode = errors.New("wa version: modo inválido")

// WAVersionService guarda a escolha do superadmin sobre a versão do cliente WhatsApp (tabela global WaVersionSettings, linha
// única). A tabela não tem RLS, então o handle normal da aplicação a lê e grava sem escopo de sistema.
type WAVersionService struct {
	db *gorm.DB
}

func NewWAVersionService(db *gorm.DB) *WAVersionService { return &WAVersionService{db: db} }

// Get devolve a escolha atual; sem linha vale o modo automático.
func (s *WAVersionService) Get() (models.WAVersionSetting, error) {
	var row models.WAVersionSetting
	err := s.db.Session(&gorm.Session{NewDB: true}).Where("id = 1").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.WAVersionSetting{ID: 1, Mode: models.WAVersionModeAuto}, nil
	}
	return row, err
}

// Save valida e grava. Em modo manual o número é obrigatório; em auto o número guardado é descartado.
func (s *WAVersionService) Save(mode, manual string) (models.WAVersionSetting, error) {
	manual = strings.TrimSpace(manual)
	row := models.WAVersionSetting{ID: 1, Mode: mode, UpdatedAt: time.Now()}
	switch mode {
	case models.WAVersionModeAuto:
	case models.WAVersionModeManual:
		if !waVersionPattern.MatchString(manual) {
			return row, fmt.Errorf("%w: %q", ErrInvalidWAVersion, manual)
		}
		row.ManualVersion = &manual
	default:
		return row, fmt.Errorf("%w: %q", ErrInvalidWAVersionMode, mode)
	}
	err := s.db.Session(&gorm.Session{NewDB: true}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"mode", "manualVersion", "updatedAt"}),
	}).Create(&row).Error
	return row, err
}

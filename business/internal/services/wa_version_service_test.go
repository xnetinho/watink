package services

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWAVersionService_SemLinhaVaiParaAutomatico(t *testing.T) {
	svc := NewWAVersionService(testutil.NewTestDB(t))
	got, err := svc.Get()
	require.NoError(t, err)
	assert.Equal(t, models.WAVersionModeAuto, got.Mode)
	assert.Nil(t, got.ManualVersion)
}

func TestWAVersionService_SalvaManualEVoltaParaAuto(t *testing.T) {
	svc := NewWAVersionService(testutil.NewTestDB(t))

	got, err := svc.Save(models.WAVersionModeManual, " 2.3000.1052747707 ")
	require.NoError(t, err)
	require.NotNil(t, got.ManualVersion)
	assert.Equal(t, "2.3000.1052747707", *got.ManualVersion)

	read, err := svc.Get()
	require.NoError(t, err)
	assert.Equal(t, models.WAVersionModeManual, read.Mode)
	require.NotNil(t, read.ManualVersion)
	assert.Equal(t, "2.3000.1052747707", *read.ManualVersion)

	_, err = svc.Save(models.WAVersionModeAuto, "2.3000.1")
	require.NoError(t, err)
	read, err = svc.Get()
	require.NoError(t, err)
	assert.Equal(t, models.WAVersionModeAuto, read.Mode)
	assert.Nil(t, read.ManualVersion, "no modo automático o número guardado é descartado")
}

func TestWAVersionService_RecusaEntradaInvalida(t *testing.T) {
	svc := NewWAVersionService(testutil.NewTestDB(t))
	for _, v := range []string{"", "abc", "2.3000", "3.3000.1", "2.3000.1052747707-alpha", "2.3000.1; DROP"} {
		_, err := svc.Save(models.WAVersionModeManual, v)
		assert.ErrorIs(t, err, ErrInvalidWAVersion, "versão %q", v)
	}
	_, err := svc.Save("qualquer", "")
	assert.ErrorIs(t, err, ErrInvalidWAVersionMode)

	got, err := svc.Get()
	require.NoError(t, err)
	assert.Equal(t, models.WAVersionModeAuto, got.Mode, "entrada recusada não grava nada")
}

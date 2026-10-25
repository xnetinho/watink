package calls_test

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/calls"
	"github.com/stretchr/testify/assert"
)

func TestIsVideoEnabled(t *testing.T) {
	t.Setenv("ENABLE_EXPERIMENTAL_VIDEO_CALLS", "")
	assert.False(t, calls.IsVideoEnabled(), "deve ser desligado por padrão")

	t.Setenv("ENABLE_EXPERIMENTAL_VIDEO_CALLS", "true")
	assert.True(t, calls.IsVideoEnabled(), "deve ativar com true")

	t.Setenv("ENABLE_EXPERIMENTAL_VIDEO_CALLS", "1")
	assert.True(t, calls.IsVideoEnabled(), "deve ativar com 1")

	t.Setenv("ENABLE_EXPERIMENTAL_VIDEO_CALLS", "false")
	assert.False(t, calls.IsVideoEnabled(), "deve desativar com false")
}

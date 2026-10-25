package telegram_test

import (
	"context"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/channels/telegram"
	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRedisForLeader struct {
	domain.RedisService
	locked bool
}

func (m *mockRedisForLeader) SetLock(key string, value string, expiration time.Duration) (bool, error) {
	if m.locked {
		return false, nil
	}
	m.locked = true
	return true, nil
}

func TestAcquireLeaderLock(t *testing.T) {
	redis := &mockRedisForLeader{}
	ctx := context.Background()

	ok, err := telegram.AcquireLeaderLock(ctx, redis, 42)
	require.NoError(t, err)
	assert.True(t, ok)

	ok2, err := telegram.AcquireLeaderLock(ctx, redis, 42)
	require.NoError(t, err)
	assert.False(t, ok2)
}

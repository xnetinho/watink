package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/stretchr/testify/assert"
)

type mockFailingRedis struct {
	domain.RedisService
}

func (m *mockFailingRedis) Ping(ctx context.Context) error {
	return errors.New("connection refused")
}

type mockPassingRedis struct {
	domain.RedisService
}

func (m *mockPassingRedis) Ping(ctx context.Context) error {
	return nil
}

func TestCheckReadiness_BrokerDown_ReturnsNotReady(t *testing.T) {
	db := testutil.NewTestDB(t)
	redis := &mockPassingRedis{}
	// RabbitMQ desconectado
	rabbit := services.NewRabbitMQProvider("amqp://guest:guest@127.0.0.1:1/")

	status := services.CheckReadiness(context.Background(), db, redis, rabbit)
	assert.Equal(t, "NOT_READY", status.Status)
	assert.True(t, status.Database)
	assert.True(t, status.Redis)
	assert.False(t, status.RabbitMQ, "RabbitMQ desconectado deve marcar rabbitmq=false")
}

func TestCheckReadiness_RedisDown_ReturnsNotReady(t *testing.T) {
	db := testutil.NewTestDB(t)
	redis := &mockFailingRedis{}
	rabbit := services.NewRabbitMQProvider("amqp://guest:guest@127.0.0.1:1/")

	status := services.CheckReadiness(context.Background(), db, redis, rabbit)
	assert.Equal(t, "NOT_READY", status.Status)
	assert.False(t, status.Redis, "Redis com falha deve marcar redis=false")
}

func TestCheckReadiness_NilDB_ReturnsNotReady(t *testing.T) {
	redis := &mockPassingRedis{}
	status := services.CheckReadiness(context.Background(), nil, redis, nil)
	assert.Equal(t, "NOT_READY", status.Status)
	assert.False(t, status.Database)
}

func TestCheckReadiness_TimeoutIsHandled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelado imediatamente

	status := services.CheckReadiness(ctx, nil, nil, nil)
	assert.Equal(t, "NOT_READY", status.Status)
}

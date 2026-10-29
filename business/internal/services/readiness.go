package services

import (
	"context"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"gorm.io/gorm"
)

type ReadinessStatus struct {
	Status   string `json:"status"`
	Database bool   `json:"database"`
	Redis    bool   `json:"redis"`
	RabbitMQ bool   `json:"rabbitmq"`
}

// CheckReadiness avalia o estado real de banco de dados, broker AMQP e cache Redis (Tarefa 8.2).
// O serviço responde como "não pronto" (HTTP 503) se qualquer uma das dependências críticas estiver fora.
func CheckReadiness(ctx context.Context, db *gorm.DB, redis domain.RedisService, rabbit *RabbitMQService) ReadinessStatus {
	ctxCheck, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	dbOK := false
	if db != nil {
		if sqlDB, err := db.DB(); err == nil && sqlDB.PingContext(ctxCheck) == nil {
			dbOK = true
		}
	}

	redisOK := false
	if redis != nil {
		if err := redis.Ping(ctxCheck); err == nil {
			redisOK = true
		}
	}

	rabbitOK := rabbit != nil && rabbit.IsConnected()

	allOK := dbOK && redisOK && rabbitOK
	status := "OK"
	if !allOK {
		status = "NOT_READY"
	}

	return ReadinessStatus{
		Status:   status,
		Database: dbOK,
		Redis:    redisOK,
		RabbitMQ: rabbitOK,
	}
}

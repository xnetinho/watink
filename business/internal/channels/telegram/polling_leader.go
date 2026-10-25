package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
)

// AcquireLeaderLock usa Redis SetNX com TTL para garantir que apenas uma instância
// execute o polling da conexão Telegram por vez (Tarefa 4.10).
func AcquireLeaderLock(ctx context.Context, redis domain.RedisService, connID int) (bool, error) {
	if redis == nil {
		return true, nil // Sem redis, assume liderança local
	}
	key := fmt.Sprintf("telegram:polling:leader:%d", connID)
	// Lock expira em 15 segundos; renovado a cada tick de polling
	acquired, err := redis.SetLock(key, "1", 15*time.Second)
	if err != nil {
		return false, err
	}
	return acquired, nil
}

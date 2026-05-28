package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type HealthService struct {
	redisClient *redis.Client
	keyPrefix   string
}

func NewHealthService(redisClient *redis.Client, keyPrefix string) *HealthService {
	return &HealthService{
		redisClient: redisClient,
		keyPrefix:   keyPrefix,
	}
}

func (s *HealthService) MarkHealthy(ctx context.Context) error {
	if s == nil || s.redisClient == nil {
		return fmt.Errorf("redis not ready")
	}

	return s.redisClient.Set(ctx, s.keyPrefix+":health", "ok", 30*time.Second).Err()
}

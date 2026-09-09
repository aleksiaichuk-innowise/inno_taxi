package redis

import (
	"context"
	"fmt"

	shared "github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context, cfg *shared.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

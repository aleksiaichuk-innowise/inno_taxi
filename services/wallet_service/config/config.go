package config

import (
	"strconv"
	"time"

	shared "github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	"github.com/joho/godotenv"
)

type Config struct {
	DbConn   shared.PostgresConfig
	Redis    *shared.RedisConfig
	HttpHost shared.HttpHostConfig
}

func Load() *Config {
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(shared.GetEnvFallback("REDIS_DB", "0"))
	if err != nil {
		redisDB = 0
	}

	return &Config{
		DbConn: shared.PostgresConfig{
			Host:     shared.GetEnvFallback("PG_WALLET_HOST", "localhost"),
			Port:     shared.GetEnvFallback("PG_WALLET_PORT", "5432"),
			Username: shared.GetEnvFallback("PG_WALLET_USER", "postgres"),
			Password: shared.GetEnvFallback("PG_WALLET_PASS", "postgres"),
			Database: shared.GetEnvFallback("PG_WALLET_DATABASE", "wallet"),

			MinConnections:        4,
			MaxConnections:        20,
			MaxConnectionLifetime: 30 * time.Minute,
			MaxIdleConnections:    5 * time.Minute,
		},
		Redis: &shared.RedisConfig{
			Addr:     shared.GetEnvFallback("REDIS_ADDR", "localhost:6379"),
			Password: shared.GetEnvFallback("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		HttpHost: shared.HttpHostConfig{
			Host: shared.GetEnvFallback("HTTP_WALLET_HOST", "localhost"),
			Port: shared.GetEnvFallback("HTTP_WALLET_PORT", "8084"),
		},
	}
}

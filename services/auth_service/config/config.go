package config

import (
	"strconv"
	"time"

	shared "github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	"github.com/joho/godotenv"
)

type Config struct {
	Redis       *shared.RedisConfig
	Host        *shared.HttpHostConfig
	JWT         *JWTConfig
	UserService *UserServiceConfig
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type UserServiceConfig struct {
	BaseURL string
}

func Load() *Config {
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(shared.GetEnvFallback("REDIS_DB", "0"))
	if err != nil {
		redisDB = 0
	}

	return &Config{
		Redis: &shared.RedisConfig{
			Addr:     shared.GetEnvFallback("REDIS_ADDR", "localhost:6379"),
			Password: shared.GetEnvFallback("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Host: &shared.HttpHostConfig{
			Host: shared.GetEnvFallback("HTTP_AUTH_HOST", "localhost"),
			Port: shared.GetEnvFallback("HTTP_AUTH_PORT", "8082"),
		},
		JWT: &JWTConfig{
			Secret:     shared.GetEnvFallback("JWT_SECRET", "taxi"),
			AccessTTL:  20 * time.Minute,
			RefreshTTL: 14 * 24 * time.Hour,
		},
		UserService: &UserServiceConfig{
			BaseURL: shared.GetEnvFallback("USER_SERVICE_BASE_URL", "http://localhost:8080"),
		},
	}
}

package config

import (
	shared "github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	"github.com/joho/godotenv"
)

type Config struct {
	Mongo *shared.MongoConfig
	Host  *shared.HttpHostConfig
	JWT   *JWTConfig
}

type JWTConfig struct {
	Secret string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Mongo: &shared.MongoConfig{
			Host:     shared.GetEnvFallback("MONGO_DS_HOST", "localhost"),
			Port:     shared.GetEnvFallback("MONGO_DS_PORT", "27017"),
			Database: shared.GetEnvFallback("MONGO_DS_DB", "driver_taxi"),
			Username: shared.GetEnvFallback("MONGO_DS_USERNAME", "taxi"),
			Password: shared.GetEnvFallback("MONGO_DS_PASSWORD", "taxi"),
		},
		Host: &shared.HttpHostConfig{
			Host: shared.GetEnvFallback("HTTP_DS_HOST", "localhost"),
			Port: shared.GetEnvFallback("HTTP_DS_PORT", "8081"),
		},
		JWT: &JWTConfig{
			Secret: shared.GetEnvFallback("JWT_SECRET", "taxi"),
		},
	}
}

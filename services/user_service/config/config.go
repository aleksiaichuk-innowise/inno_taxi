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
	Secret string `env:"JWT_SECRET"`
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Mongo: &shared.MongoConfig{
			Host:     shared.GetEnvFallback("MONGO_US_HOST", "localhost"),
			Port:     shared.GetEnvFallback("MONGO_US_PORT", "27017"),
			Database: shared.GetEnvFallback("MONGO_US_DB", "test"),
			Username: shared.GetEnvFallback("MONGO_US_USERNAME", "taxi"),
			Password: shared.GetEnvFallback("MONGO_US_PASSWORD", "taxi"),
		},
		Host: &shared.HttpHostConfig{
			Host: shared.GetEnvFallback("HTTP_US_HOST", "localhost"),
			Port: shared.GetEnvFallback("HTTP_US_PORT", "8080"),
		},
		JWT: &JWTConfig{
			Secret: shared.GetEnvFallback("JWT_SECRET", "taxi"),
		},
	}
}

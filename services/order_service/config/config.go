package config

import (
	"strings"
	"time"

	shared "github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	"github.com/joho/godotenv"
)

type Config struct {
	DbConn    shared.PostgresConfig
	EsConn    shared.ElasticsearchConfig
	KafkaConf shared.KafkaConfig
	Grpc      *shared.GrpcServerConfig
	JWT       *JWTConfig
}

type JWTConfig struct {
	Secret string `env:"JWT_SECRET"`
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		DbConn: shared.PostgresConfig{
			Host:     shared.GetEnvFallback("PG_ORDER_HOST", "localhost"),
			Port:     shared.GetEnvFallback("PG_ORDER_PORT", "5432"),
			Username: shared.GetEnvFallback("PG_ORDER_USER", "postgres"),
			Password: shared.GetEnvFallback("PG_ORDER_PASS", "postgres"),
			Database: shared.GetEnvFallback("PG_ORDER_DATABASE", "order"),

			MinConnections:        4,
			MaxConnections:        20,
			MaxConnectionLifetime: 30 * time.Minute,
			MaxIdleConnections:    5 * time.Minute,
		},
		EsConn: shared.ElasticsearchConfig{
			Addresses: strings.Split(shared.GetEnvFallback("ES_ORDER_HOST", "localhost:9200"), ","),
			Username:  shared.GetEnvFallback("ES_ORDER_USER", "elastic"),
			Password:  shared.GetEnvFallback("ES_ORDER_PASS", "elastic"),

			MaxIdleConnsPerHost: 20,
			Timeout:             10 * time.Second,
		},
		KafkaConf: shared.KafkaConfig{
			Brokers:  strings.Split(shared.GetEnvFallback("KAFKA_BROKERS", "localhost:9092"), ","),
			Username: shared.GetEnvFallback("KAFKA_USERNAME", ""),
			Password: shared.GetEnvFallback("KAFKA_PASSWORD", ""),
		},
		Grpc: &shared.GrpcServerConfig{
			Host: shared.GetEnvFallback("GRPC_ORDER_HOST", "localhost"),
			Port: shared.GetEnvFallback("GRPC_ORDER_PORT", "9082"),

			MaxConnectionIdle: 15 * time.Minute,
			Timeout:           10 * time.Second,
		},
		JWT: &JWTConfig{
			Secret: shared.GetEnvFallback("JWT_SECRET", "taxi"),
		},
	}
}

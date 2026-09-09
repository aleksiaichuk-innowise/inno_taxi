package config

import (
	"strings"

	shared "github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	"github.com/joho/godotenv"
)

type Config struct {
	Kafka      shared.KafkaConfig
	ClickHouse ClickHouseConfig
	HttpHost   shared.HttpHostConfig
}

type ClickHouseConfig struct {
	Addr     string
	Database string
	Username string
	Password string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Kafka: shared.KafkaConfig{
			Brokers:  strings.Split(shared.GetEnvFallback("KAFKA_BROKERS", "localhost:9092"), ","),
			Username: shared.GetEnvFallback("KAFKA_USERNAME", ""),
			Password: shared.GetEnvFallback("KAFKA_PASSWORD", ""),
		},
		ClickHouse: ClickHouseConfig{
			Addr:     shared.GetEnvFallback("CLICKHOUSE_ADDR", "localhost:9000"),
			Database: shared.GetEnvFallback("CLICKHOUSE_DATABASE", "analytics"),
			Username: shared.GetEnvFallback("CLICKHOUSE_USERNAME", "default"),
			Password: shared.GetEnvFallback("CLICKHOUSE_PASSWORD", ""),
		},
		HttpHost: shared.HttpHostConfig{
			Host: shared.GetEnvFallback("HTTP_ANALYTIC_HOST", "localhost"),
			Port: shared.GetEnvFallback("HTTP_ANALYTIC_PORT", "8085"),
		},
	}
}

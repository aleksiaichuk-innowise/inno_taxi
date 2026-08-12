package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Mongo *MongoConfig
	Host  *HostConfig
}
type MongoConfig struct {
	Host     string `env:"MONGO_HOST" envDefault:"localhost"`
	Port     string `env:"MONGO_PORT" envDefault:"27017"`
	Database string `env:"MONGO_DB" envDefault:"test"`
	Username string `env:"MONGO_USERNAME" envDefault:"taxi"`
	Password string `env:"MONGO_PASSWORD" envDefault:"taxi"`
}
type HostConfig struct {
	Host string `env:"HTTP_HOST" envDefault:"localhost"`
	Port string `env:"HTTP_PORT" envDefault:"8080"`
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Mongo: &MongoConfig{
			Host:     getEnv("MONGO_HOST", "localhost"),
			Port:     getEnv("MONGO_PORT", "27017"),
			Database: getEnv("MONGO_DB", "test"),
			Username: getEnv("MONGO_USERNAME", "taxi"),
			Password: getEnv("MONGO_PASSWORD", "taxi"),
		},
		Host: &HostConfig{
			Host: getEnv("HTTP_HOST", "localhost"),
			Port: getEnv("HTTP_PORT", "8080"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

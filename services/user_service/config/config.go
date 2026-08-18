package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Mongo *MongoConfig
	Host  *HostConfig
	JWT   *JWTConfig
}
type MongoConfig struct {
	Host     string `env:"MONGO_HOST"`
	Port     string `env:"MONGO_PORT"`
	Database string `env:"MONGO_DB"`
	Username string `env:"MONGO_USERNAME"`
	Password string `env:"MONGO_PASSWORD"`
}
type HostConfig struct {
	Host string `env:"HTTP_HOST"`
	Port string `env:"HTTP_PORT"`
}

type JWTConfig struct {
	Secret string `env:"JWT_SECRET"`
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
		JWT: &JWTConfig{
			Secret: getEnv("JWT_SECRET", "taxi"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

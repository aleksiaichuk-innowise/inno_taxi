package config

type HttpHostConfig struct {
	Host string `env:"HTTP_HOST"`
	Port string `env:"HTTP_PORT"`
}

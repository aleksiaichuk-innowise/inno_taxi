package config

type MongoConfig struct {
	Host     string `env:"MONGO_HOST"`
	Port     string `env:"MONGO_PORT"`
	Database string `env:"MONGO_DB"`
	Username string `env:"MONGO_USERNAME"`
	Password string `env:"MONGO_PASSWORD"`
}

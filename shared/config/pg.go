package config

import "time"

type PostgresConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string

	MaxConnections        int32
	MinConnections        int32
	MaxConnectionLifetime time.Duration
	MaxIdleConnections    time.Duration
}

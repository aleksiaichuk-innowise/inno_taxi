package config

import "time"

type GrpcServerConfig struct {
	Host              string
	Port              string
	MaxConnectionIdle time.Duration
	Timeout           time.Duration
}

type GrpcClientConfig struct {
	TargetAddress string
	DialTimeout   time.Duration
	Insecure      bool
}

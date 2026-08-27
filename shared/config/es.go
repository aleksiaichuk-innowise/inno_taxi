package config

import "time"

type ElasticsearchConfig struct {
	Addresses []string
	Username  string
	Password  string
	APIKey    string

	MaxIdleConnsPerHost int
	Timeout             time.Duration
}

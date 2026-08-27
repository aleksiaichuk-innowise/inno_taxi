package postgres

import (
	"context"
	"fmt"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPgPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
	)

	poolCfg, err := pgxpool.ParseConfig(connStr)

	if err != nil {
		return nil, err
	}

	poolCfg.MaxConns = cfg.MaxConnections
	poolCfg.MinConns = cfg.MinConnections
	poolCfg.MaxConnLifetime = cfg.MaxConnectionLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxIdleConnections

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)

	if err != nil {
		return nil, fmt.Errorf("pgxpool.NewWithConfig: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pgxpool.Ping: %w", err)
	}

	return pool, nil
}

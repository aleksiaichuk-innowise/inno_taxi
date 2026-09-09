package postgres

import (
	"context"
	"fmt"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func NewDB(ctx context.Context, cfg config.PostgresConfig) (*sqlx.DB, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
	)

	db, err := sqlx.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("sqlx.Open: %w", err)
	}

	db.SetMaxOpenConns(int(cfg.MaxConnections))
	db.SetMaxIdleConns(int(cfg.MinConnections))
	db.SetConnMaxLifetime(cfg.MaxConnectionLifetime)
	db.SetConnMaxIdleTime(cfg.MaxIdleConnections)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	return db, nil
}

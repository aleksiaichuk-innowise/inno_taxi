package clickhouse

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/config"
)

//go:embed schema/create_order_events.sql
var schema string

func New(ctx context.Context, cfg config.ClickHouseConfig) (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse connection: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}

	if err := conn.Exec(ctx, schema); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return conn, nil
}

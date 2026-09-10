package clickhouse

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/config"
)

//go:embed schema/create_order_events.sql
var orderEventsSchema string

//go:embed schema/create_user_registration_events.sql
var userRegistrationEventsSchema string

//go:embed schema/create_driver_ratings.sql
var driverRatingsSchema string

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

	if err := conn.Exec(ctx, orderEventsSchema); err != nil {
		return nil, fmt.Errorf("apply order_events schema: %w", err)
	}
	if err := conn.Exec(ctx, userRegistrationEventsSchema); err != nil {
		return nil, fmt.Errorf("apply user_registration_events schema: %w", err)
	}
	if err := conn.Exec(ctx, driverRatingsSchema); err != nil {
		return nil, fmt.Errorf("apply driver_ratings schema: %w", err)
	}

	return conn, nil
}

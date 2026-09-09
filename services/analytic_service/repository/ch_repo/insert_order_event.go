package ch_repo

import (
	"context"
	"fmt"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

func (r *ClickHouseRepository) InsertOrderEvent(ctx context.Context, evt service_dto.OrderEvent) error {
	const query = `
		INSERT INTO order_events (order_id, user_id, taxi_type, status, order_created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	if err := r.conn.Exec(ctx, query, evt.OrderID, evt.UserID, evt.TaxiType, evt.Status, evt.CreatedAt); err != nil {
		return fmt.Errorf("insert order event: %w", err)
	}
	return nil
}

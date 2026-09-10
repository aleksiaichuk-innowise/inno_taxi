package ch_repo

import (
	"context"
	"fmt"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

func (r *ClickHouseRepository) InsertDriverRating(ctx context.Context, evt service_dto.DriverRatingEvent) error {
	const query = `
		INSERT INTO driver_ratings (order_id, driver_id, rating, comment, rated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	if err := r.conn.Exec(ctx, query, evt.OrderID, evt.DriverID, evt.Rating, evt.Comment, evt.RatedAt); err != nil {
		return fmt.Errorf("insert driver rating: %w", err)
	}
	return nil
}

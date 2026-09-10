package ch_repo

import (
	"context"
	"fmt"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

// GetDriverRatingStats averages over the driver's most recent 20 rated
// trips only - README says "last 20 trips", not "all trips" - via a
// subquery that orders by rated_at DESC and limits before aggregating.
// FINAL guards against double-counting a Kafka-redelivered rating the
// same way GetOrderStats does for order_events.
func (r *ClickHouseRepository) GetDriverRatingStats(ctx context.Context, driverID string) (service_dto.DriverRatingStats, error) {
	const query = `
		SELECT avg(rating), count()
		FROM (
			SELECT rating
			FROM driver_ratings FINAL
			WHERE driver_id = ?
			ORDER BY rated_at DESC
			LIMIT 20
		)
	`

	row := r.conn.QueryRow(ctx, query, driverID)

	var avg float64
	var count uint64
	if err := row.Scan(&avg, &count); err != nil {
		return service_dto.DriverRatingStats{}, fmt.Errorf("query driver rating stats: %w", err)
	}
	if count == 0 {
		return service_dto.DriverRatingStats{}, nil
	}

	return service_dto.DriverRatingStats{Average: avg, Count: int64(count)}, nil
}

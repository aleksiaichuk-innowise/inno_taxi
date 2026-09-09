package ch_repo

import (
	"context"
	"fmt"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

// GetOrderStats queries FINAL so a row that ReplacingMergeTree hasn't
// merged away yet (a Kafka redelivery arriving between merges) doesn't
// get double-counted - see the design doc's "known trade-off" on cost
// vs. correctness for this.
func (r *ClickHouseRepository) GetOrderStats(ctx context.Context, from, to time.Time) (service_dto.OrderStats, error) {
	const query = `
		SELECT status, taxi_type, count() AS cnt
		FROM order_events FINAL
		WHERE order_created_at >= ? AND order_created_at < ?
		GROUP BY status, taxi_type
	`

	rows, err := r.conn.Query(ctx, query, from, to)
	if err != nil {
		return service_dto.OrderStats{}, fmt.Errorf("query order stats: %w", err)
	}
	defer rows.Close()

	stats := service_dto.OrderStats{
		CountsByStatus:   map[string]int64{},
		CountsByTaxiType: map[string]int64{},
	}
	for rows.Next() {
		var status, taxiType string
		var count uint64
		if err := rows.Scan(&status, &taxiType, &count); err != nil {
			return service_dto.OrderStats{}, fmt.Errorf("scan order stats row: %w", err)
		}
		stats.CountsByStatus[status] += int64(count)
		stats.CountsByTaxiType[taxiType] += int64(count)
		stats.TotalOrders += int64(count)
	}
	if err := rows.Err(); err != nil {
		return service_dto.OrderStats{}, fmt.Errorf("iterate order stats rows: %w", err)
	}

	return stats, nil
}

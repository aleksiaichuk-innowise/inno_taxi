package ch_repo

import (
	"context"
	"fmt"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

func (r *ClickHouseRepository) GetDailyOrderCounts(ctx context.Context, from, to time.Time) ([]service_dto.DailyOrderCount, error) {
	const query = `
		SELECT toDate(order_created_at) AS d, count() AS cnt
		FROM order_events FINAL
		WHERE order_created_at >= ? AND order_created_at < ?
		GROUP BY d
		ORDER BY d
	`

	rows, err := r.conn.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("query daily order counts: %w", err)
	}
	defer rows.Close()

	var results []service_dto.DailyOrderCount
	for rows.Next() {
		var day time.Time
		var count uint64
		if err := rows.Scan(&day, &count); err != nil {
			return nil, fmt.Errorf("scan daily order count row: %w", err)
		}
		results = append(results, service_dto.DailyOrderCount{
			Date:  day.Format("2006-01-02"),
			Count: int64(count),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily order count rows: %w", err)
	}

	return results, nil
}

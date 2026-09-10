package pg_repo

import (
	"context"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/jackc/pgx/v5"
)

// ListOrdersByUser returns orders where the caller is either the rider or
// the assigned driver, most recent first - see the design doc for why one
// query serves both sides of the trip.
func (r PgRepository) ListOrdersByUser(ctx context.Context, userID string, limit, offset int32) ([]service_dto.Order, int64, error) {
	const listQuery = `
		SELECT id, user_id, driver_id, taxi_type, start_lat, start_lng, destination_lat, destination_lng, status, price_minor_units, rating, comment, created_at, updated_at
		FROM orders
		WHERE user_id = @user_id OR driver_id = @user_id
		ORDER BY created_at DESC
		LIMIT @limit OFFSET @offset
	`
	rows, err := r.pool.Query(ctx, listQuery, pgx.NamedArgs{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list orders query: %w", err)
	}

	repoRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[repo_entity.Order])
	if err != nil {
		return nil, 0, fmt.Errorf("list orders scan: %w", err)
	}

	orders := make([]service_dto.Order, len(repoRows))
	for i, row := range repoRows {
		orders[i] = row.ToDomain()
	}

	const countQuery = `
		SELECT count(*) FROM orders WHERE user_id = @user_id OR driver_id = @user_id
	`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, pgx.NamedArgs{"user_id": userID}).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	return orders, total, nil
}

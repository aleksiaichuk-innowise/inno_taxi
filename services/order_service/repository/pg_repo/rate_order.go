package pg_repo

import (
	"context"
	"errors"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
	"github.com/jackc/pgx/v5"
)

// RateOrder's "AND rating IS NULL" is the load-bearing part: it's what
// catches a genuine double-submit race between two concurrent RateTrip
// calls for the same order, not just the service layer's earlier read.
func (r PgRepository) RateOrder(ctx context.Context, id string, rating int32, comment *string) (service_dto.Order, error) {
	const query = `
		UPDATE orders
		SET rating = @rating, comment = @comment, updated_at = now()
		WHERE id = @id AND rating IS NULL
		RETURNING id, user_id, driver_id, taxi_type, start_lat, start_lng, destination_lat, destination_lng, status, price_minor_units, rating, comment, created_at, updated_at
	`
	rows, err := r.pool.Query(ctx, query, pgx.NamedArgs{
		"id":      id,
		"rating":  rating,
		"comment": comment,
	})
	if err != nil {
		return service_dto.Order{}, fmt.Errorf("rate order query: %w", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[repo_entity.Order])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service_dto.Order{}, errorsx.ErrOrderAlreadyRated
		}
		return service_dto.Order{}, fmt.Errorf("rate order scan: %w", err)
	}

	return row.ToDomain(), nil
}

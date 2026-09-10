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

func (r PgRepository) GetOrderByID(ctx context.Context, id string) (service_dto.Order, error) {
	const query = `
		SELECT id, user_id, driver_id, taxi_type, start_lat, start_lng, destination_lat, destination_lng, status, price_minor_units, rating, comment, created_at, updated_at
		FROM orders
		WHERE id = @id
	`
	rows, err := r.pool.Query(ctx, query, pgx.NamedArgs{"id": id})
	if err != nil {
		return service_dto.Order{}, fmt.Errorf("get order query: %w", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[repo_entity.Order])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service_dto.Order{}, errorsx.ErrOrderNotFound
		}
		return service_dto.Order{}, fmt.Errorf("get order scan: %w", err)
	}

	return row.ToDomain(), nil
}

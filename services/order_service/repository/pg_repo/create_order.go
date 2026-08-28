package pg_repo

import (
	"context"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/jackc/pgx/v5"
)

func (r PgRepository) CreateOrder(ctx context.Context, input service_dto.CreateOrderInput) (service_dto.Order, error) {
	const query = `
		INSERT INTO orders (user_id, taxi_type, start_lat, start_lng, destination_lat, destination_lng)
		VALUES (@user_id, @taxi_type, @start_lat, @start_lng, @destination_lat, @destination_lng)
		RETURNING id, user_id, driver_id, taxi_type, start_lat, start_lng, destination_lat, destination_lng, status, price_minor_units, created_at, updated_at
	`
	rows, err := r.pool.Query(ctx, query, pgx.NamedArgs{
		"user_id":         input.UserID,
		"taxi_type":       string(input.TaxiType),
		"start_lat":       input.Start.Lat,
		"start_lng":       input.Start.Lng,
		"destination_lat": input.Destination.Lat,
		"destination_lng": input.Destination.Lng,
	})
	if err != nil {
		return service_dto.Order{}, fmt.Errorf("create order query: %w", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[repo_entity.Order])
	if err != nil {
		return service_dto.Order{}, fmt.Errorf("create order scan: %w", err)
	}

	return row.ToDomain(), nil
}

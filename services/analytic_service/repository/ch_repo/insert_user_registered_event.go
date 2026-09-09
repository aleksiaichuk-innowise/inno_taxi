package ch_repo

import (
	"context"
	"fmt"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

func (r *ClickHouseRepository) InsertUserRegisteredEvent(ctx context.Context, evt service_dto.UserRegisteredEvent) error {
	const query = `
		INSERT INTO user_registration_events (user_id, name, email, phone, role, registered_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	if err := r.conn.Exec(ctx, query, evt.UserID, evt.Name, evt.Email, evt.Phone, evt.Role, evt.RegisteredAt); err != nil {
		return fmt.Errorf("insert user registered event: %w", err)
	}
	return nil
}

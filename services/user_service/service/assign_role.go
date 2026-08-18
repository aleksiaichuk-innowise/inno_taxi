package service

import (
	"context"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
)

func (s UserService) AssignRole(ctx context.Context, userID string, role serviceEntity.Role) error {
	if !role.IsValid() {
		return errorsx.ErrInvalidRole
	}

	return s.userRepo.AddRole(ctx, userID, string(role))
}

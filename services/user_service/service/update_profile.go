package service

import (
	"context"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
)

func (s UserService) UpdateProfile(ctx context.Context, id string, input serviceEntity.ProfileInput) (*serviceEntity.User, error) {
	return s.userRepo.UpdateProfile(ctx, id, input.Name, input.Email, input.Phone)
}

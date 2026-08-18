package service

import (
	"context"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
)

func (s UserService) GetProfile(ctx context.Context, id string) (*serviceEntity.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

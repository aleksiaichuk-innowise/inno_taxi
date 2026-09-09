package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/service"
)

func (s *AuthService) Login(ctx context.Context, login, password string) (service_dto.TokenPair, error) {
	user, err := s.userService.VerifyCredentials(ctx, login, password)
	if err != nil {
		return service_dto.TokenPair{}, err
	}
	return s.issueTokenPair(ctx, user.ID, user.Roles)
}

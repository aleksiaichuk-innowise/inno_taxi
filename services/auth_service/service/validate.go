package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/errorsx"
	"github.com/golang-jwt/jwt/v5"
)

func (s *AuthService) Validate(ctx context.Context, accessToken string) (service_dto.AccessClaims, error) {
	claims := &service_dto.AccessClaims{}
	token, err := jwt.ParseWithClaims(accessToken, claims, s.keyFunc)
	if err != nil || !token.Valid {
		return service_dto.AccessClaims{}, errorsx.ErrInvalidToken
	}

	active, err := s.sessions.AccessSessionExists(ctx, claims.ID)
	if err != nil {
		return service_dto.AccessClaims{}, err
	}
	if !active {
		return service_dto.AccessClaims{}, errorsx.ErrInvalidToken
	}

	return *claims, nil
}

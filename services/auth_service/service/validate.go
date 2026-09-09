package service

import (
	"context"
	"slices"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/errorsx"
	"github.com/golang-jwt/jwt/v5"
)

// Validate checks that accessToken is a currently-active session and, when
// requiredRole is non-empty, that its claims carry that role - the gateway
// declares which role a route needs (per location, in nginx.conf), Auth
// Service is just where the check against the token's claims happens.
func (s *AuthService) Validate(ctx context.Context, accessToken, requiredRole string) (service_dto.AccessClaims, error) {
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

	if requiredRole != "" && !slices.Contains(claims.Roles, requiredRole) {
		return service_dto.AccessClaims{}, errorsx.ErrForbidden
	}

	return *claims, nil
}

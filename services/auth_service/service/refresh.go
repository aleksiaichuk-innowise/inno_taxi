package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/errorsx"
	"github.com/golang-jwt/jwt/v5"
)

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (service_dto.TokenPair, error) {
	claims := &service_dto.RefreshClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, s.keyFunc)
	if err != nil || !token.Valid {
		return service_dto.TokenPair{}, errorsx.ErrInvalidToken
	}

	active, err := s.sessions.RefreshSessionExists(ctx, claims.ID)
	if err != nil {
		return service_dto.TokenPair{}, err
	}
	if !active {
		return service_dto.TokenPair{}, errorsx.ErrInvalidToken
	}

	// Rotate: retire the old session the moment a new pair is issued, so a
	// stolen refresh token can be replayed at most once.
	if err := s.sessions.DeleteAccessSession(ctx, claims.ID); err != nil {
		return service_dto.TokenPair{}, err
	}
	if err := s.sessions.DeleteRefreshSession(ctx, claims.ID); err != nil {
		return service_dto.TokenPair{}, err
	}

	return s.issueTokenPair(ctx, claims.UserID, claims.Roles)
}

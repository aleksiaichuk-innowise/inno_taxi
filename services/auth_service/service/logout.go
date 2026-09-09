package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/service"
	"github.com/golang-jwt/jwt/v5"
)

// Logout is best-effort and idempotent: a token that fails to parse simply
// contributes no session to delete rather than being treated as an error -
// the caller's goal ("make sure these tokens stop working") already holds
// for a token that was never valid to begin with.
func (s *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if sid, ok := s.sidFromAccessToken(accessToken); ok {
		if err := s.sessions.DeleteAccessSession(ctx, sid); err != nil {
			return err
		}
	}
	if sid, ok := s.sidFromRefreshToken(refreshToken); ok {
		if err := s.sessions.DeleteRefreshSession(ctx, sid); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthService) sidFromAccessToken(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}
	claims := &service_dto.AccessClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, s.keyFunc)
	if err != nil || !token.Valid {
		return "", false
	}
	return claims.ID, true
}

func (s *AuthService) sidFromRefreshToken(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}
	claims := &service_dto.RefreshClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, s.keyFunc)
	if err != nil || !token.Valid {
		return "", false
	}
	return claims.ID, true
}

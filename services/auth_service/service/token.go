package service

import (
	"context"
	"fmt"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (s *AuthService) keyFunc(token *jwt.Token) (any, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	return []byte(s.secret), nil
}

func (s *AuthService) issueTokenPair(ctx context.Context, userID string, roles []string) (service_dto.TokenPair, error) {
	sid := uuid.NewString()
	now := time.Now()

	accessClaims := service_dto.AccessClaims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        sid,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}
	refreshClaims := service_dto.RefreshClaims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        sid,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.secret))
	if err != nil {
		return service_dto.TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.secret))
	if err != nil {
		return service_dto.TokenPair{}, fmt.Errorf("sign refresh token: %w", err)
	}

	if err := s.sessions.SaveAccessSession(ctx, sid, s.accessTTL); err != nil {
		return service_dto.TokenPair{}, fmt.Errorf("save access session: %w", err)
	}
	if err := s.sessions.SaveRefreshSession(ctx, sid, s.refreshTTL); err != nil {
		return service_dto.TokenPair{}, fmt.Errorf("save refresh session: %w", err)
	}

	return service_dto.TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

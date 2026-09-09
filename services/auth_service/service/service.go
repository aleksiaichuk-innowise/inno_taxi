package service

import (
	"context"
	"time"

	gateway_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/gateway"
)

type SessionStore interface {
	SaveAccessSession(ctx context.Context, sid string, ttl time.Duration) error
	SaveRefreshSession(ctx context.Context, sid string, ttl time.Duration) error
	AccessSessionExists(ctx context.Context, sid string) (bool, error)
	RefreshSessionExists(ctx context.Context, sid string) (bool, error)
	DeleteAccessSession(ctx context.Context, sid string) error
	DeleteRefreshSession(ctx context.Context, sid string) error
}

type UserServiceGateway interface {
	VerifyCredentials(ctx context.Context, login, password string) (gateway_dto.UserInfo, error)
}

type AuthService struct {
	sessions    SessionStore
	userService UserServiceGateway

	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(sessions SessionStore, userService UserServiceGateway, secret string, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		sessions:    sessions,
		userService: userService,
		secret:      secret,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}
}

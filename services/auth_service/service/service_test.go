package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	gateway_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/entity/gateway"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/errorsx"
)

type fakeSessionStore struct {
	mu       sync.Mutex
	access   map[string]bool
	refresh  map[string]bool
	saveErr  error
	existErr error
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{access: map[string]bool{}, refresh: map[string]bool{}}
}

func (f *fakeSessionStore) SaveAccessSession(_ context.Context, sid string, _ time.Duration) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.access[sid] = true
	return nil
}

func (f *fakeSessionStore) SaveRefreshSession(_ context.Context, sid string, _ time.Duration) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refresh[sid] = true
	return nil
}

func (f *fakeSessionStore) AccessSessionExists(_ context.Context, sid string) (bool, error) {
	if f.existErr != nil {
		return false, f.existErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.access[sid], nil
}

func (f *fakeSessionStore) RefreshSessionExists(_ context.Context, sid string) (bool, error) {
	if f.existErr != nil {
		return false, f.existErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refresh[sid], nil
}

func (f *fakeSessionStore) DeleteAccessSession(_ context.Context, sid string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.access, sid)
	return nil
}

func (f *fakeSessionStore) DeleteRefreshSession(_ context.Context, sid string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.refresh, sid)
	return nil
}

type fakeUserServiceGateway struct {
	user *gateway_dto.UserInfo
	err  error
}

func (f *fakeUserServiceGateway) VerifyCredentials(_ context.Context, _, _ string) (gateway_dto.UserInfo, error) {
	if f.err != nil {
		return gateway_dto.UserInfo{}, f.err
	}
	return *f.user, nil
}

func newTestAuthService(sessions *fakeSessionStore, userService *fakeUserServiceGateway) *AuthService {
	return NewAuthService(sessions, userService, "test-secret", 20*time.Minute, 14*24*time.Hour)
}

func TestLogin_Success(t *testing.T) {
	sessions := newFakeSessionStore()
	userService := &fakeUserServiceGateway{user: &gateway_dto.UserInfo{ID: "user-1", Roles: []string{"user"}}}
	svc := newTestAuthService(sessions, userService)

	pair, err := svc.Login(context.Background(), "user@example.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected non-empty tokens, got %+v", pair)
	}

	claims, err := svc.Validate(context.Background(), pair.AccessToken, "")
	if err != nil {
		t.Fatalf("expected freshly issued access token to validate, got %v", err)
	}
	if claims.UserID != "user-1" || len(claims.Roles) != 1 || claims.Roles[0] != "user" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	sessions := newFakeSessionStore()
	userService := &fakeUserServiceGateway{err: errorsx.ErrInvalidCredentials}
	svc := newTestAuthService(sessions, userService)

	_, err := svc.Login(context.Background(), "user@example.com", "wrong")
	if !errors.Is(err, errorsx.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidate_TokenSignedWithDifferentSecret(t *testing.T) {
	sessions := newFakeSessionStore()
	userService := &fakeUserServiceGateway{user: &gateway_dto.UserInfo{ID: "user-1", Roles: []string{"user"}}}
	otherSvc := NewAuthService(sessions, userService, "different-secret", time.Minute, time.Hour)

	pair, err := otherSvc.Login(context.Background(), "u", "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := newTestAuthService(sessions, userService)
	if _, err := svc.Validate(context.Background(), pair.AccessToken, ""); !errors.Is(err, errorsx.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for a token signed with a different secret, got %v", err)
	}
}

func TestValidate_RevokedSession(t *testing.T) {
	sessions := newFakeSessionStore()
	userService := &fakeUserServiceGateway{user: &gateway_dto.UserInfo{ID: "user-1", Roles: []string{"user"}}}
	svc := newTestAuthService(sessions, userService)

	pair, err := svc.Login(context.Background(), "u", "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.Logout(context.Background(), pair.AccessToken, pair.RefreshToken); err != nil {
		t.Fatalf("unexpected logout error: %v", err)
	}

	_, err = svc.Validate(context.Background(), pair.AccessToken, "")
	if !errors.Is(err, errorsx.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken after logout, got %v", err)
	}
}

func TestRefresh_RotatesSession(t *testing.T) {
	sessions := newFakeSessionStore()
	userService := &fakeUserServiceGateway{user: &gateway_dto.UserInfo{ID: "user-1", Roles: []string{"user"}}}
	svc := newTestAuthService(sessions, userService)

	original, err := svc.Login(context.Background(), "u", "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rotated, err := svc.Refresh(context.Background(), original.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected refresh error: %v", err)
	}
	if rotated.AccessToken == original.AccessToken || rotated.RefreshToken == original.RefreshToken {
		t.Fatal("expected refresh to issue a brand new token pair")
	}

	// The old refresh token must not be usable a second time (rotation).
	if _, err := svc.Refresh(context.Background(), original.RefreshToken); !errors.Is(err, errorsx.ErrInvalidToken) {
		t.Fatalf("expected old refresh token to be invalid after rotation, got %v", err)
	}

	// The new pair must work.
	if _, err := svc.Validate(context.Background(), rotated.AccessToken, ""); err != nil {
		t.Fatalf("expected new access token to validate, got %v", err)
	}
}

func TestValidate_RequiredRole(t *testing.T) {
	sessions := newFakeSessionStore()
	userService := &fakeUserServiceGateway{user: &gateway_dto.UserInfo{ID: "user-1", Roles: []string{"user", "admin"}}}
	svc := newTestAuthService(sessions, userService)

	pair, err := svc.Login(context.Background(), "u", "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := svc.Validate(context.Background(), pair.AccessToken, "admin"); err != nil {
		t.Fatalf("expected token carrying admin role to satisfy requiredRole=admin, got %v", err)
	}

	if _, err := svc.Validate(context.Background(), pair.AccessToken, "analyst"); !errors.Is(err, errorsx.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a role the token doesn't carry, got %v", err)
	}
}

func TestRefresh_UnknownToken(t *testing.T) {
	sessions := newFakeSessionStore()
	svc := newTestAuthService(sessions, &fakeUserServiceGateway{})

	_, err := svc.Refresh(context.Background(), "not-a-jwt")
	if !errors.Is(err, errorsx.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestLogout_IdempotentOnGarbageInput(t *testing.T) {
	sessions := newFakeSessionStore()
	svc := newTestAuthService(sessions, &fakeUserServiceGateway{})

	if err := svc.Logout(context.Background(), "", ""); err != nil {
		t.Fatalf("expected logout with empty tokens to succeed, got %v", err)
	}
	if err := svc.Logout(context.Background(), "garbage", "garbage"); err != nil {
		t.Fatalf("expected logout with unparseable tokens to succeed, got %v", err)
	}
}

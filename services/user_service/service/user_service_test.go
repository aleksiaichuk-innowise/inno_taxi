package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
)

type fakeUserRepository struct {
	user *serviceEntity.User
	err  error
}

func (f *fakeUserRepository) CreateUser(_ context.Context, _ *serviceEntity.User) (*serviceEntity.User, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeUserRepository) FindByLogin(_ context.Context, _ string) (*serviceEntity.User, error) {
	return f.user, f.err
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return string(hash)
}

func TestUserService_VerifyCredentials_Success(t *testing.T) {
	repo := &fakeUserRepository{
		user: &serviceEntity.User{
			ID:           "1",
			Email:        "user@example.com",
			PasswordHash: hashPassword(t, "correct-password"),
			Roles:        []serviceEntity.Role{serviceEntity.RoleUser},
		},
	}
	svc := NewUserService(repo)

	got, err := svc.VerifyCredentials(context.Background(), "user@example.com", "correct-password")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("got user ID %q, want %q", got.ID, "1")
	}
}

func TestUserService_VerifyCredentials_WrongPassword(t *testing.T) {
	repo := &fakeUserRepository{
		user: &serviceEntity.User{
			ID:           "1",
			Email:        "user@example.com",
			PasswordHash: hashPassword(t, "correct-password"),
		},
	}
	svc := NewUserService(repo)

	_, err := svc.VerifyCredentials(context.Background(), "user@example.com", "wrong-password")

	if !errors.Is(err, errorsx.ErrInvalidCredentials) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidCredentials)
	}
}

func TestUserService_VerifyCredentials_UserNotFound(t *testing.T) {
	repo := &fakeUserRepository{err: errorsx.ErrUserNotFound}
	svc := NewUserService(repo)

	_, err := svc.VerifyCredentials(context.Background(), "missing@example.com", "any-password")

	if !errors.Is(err, errorsx.ErrInvalidCredentials) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidCredentials)
	}
}

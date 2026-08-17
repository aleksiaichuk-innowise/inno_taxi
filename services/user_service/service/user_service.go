package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *serviceEntity.User) (*serviceEntity.User, error)
	FindByLogin(ctx context.Context, login string) (*serviceEntity.User, error)
}

type UserService struct {
	userRepo UserRepository
}

func NewUserService(userRepo UserRepository) *UserService {
	return &UserService{userRepo}
}

func (s UserService) CreateUser(ctx context.Context, input serviceEntity.RegisterInput) (*serviceEntity.User, error) {
	role := serviceEntity.Role(strings.ToLower(string(input.Role)))
	if role != serviceEntity.RoleUser && role != serviceEntity.RoleDriver {
		return nil, errorsx.ErrInvalidRole
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &serviceEntity.User{
		Name:         input.Name,
		Email:        input.Email,
		Phone:        input.Phone,
		PasswordHash: string(hash),
		Roles:        []serviceEntity.Role{role},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := s.userRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s UserService) VerifyCredentials(ctx context.Context, login string, password string) (*serviceEntity.User, error) {
	u, err := s.userRepo.FindByLogin(ctx, login)
	if err != nil {
		return nil, errorsx.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, errorsx.ErrInvalidCredentials
	}

	return u, nil
}

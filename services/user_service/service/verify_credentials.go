package service

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/errorsx"
)

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

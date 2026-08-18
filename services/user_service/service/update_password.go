package service

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
)

func (s UserService) UpdatePassword(ctx context.Context, id string, currentPasswd string, newPasswd string) error {
	usr, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usr.PasswordHash), []byte(currentPasswd)); err != nil {
		return errorsx.ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPasswd), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.userRepo.SetPassword(ctx, id, string(hash))
}

package service

import (
	"context"
	"errors"
	"testing"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/errorsx"
)

type createUserFakeRepo struct {
	fakeUserRepository
	created *serviceEntity.User
	err     error
}

func (f *createUserFakeRepo) CreateUser(_ context.Context, user *serviceEntity.User) (*serviceEntity.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.created != nil {
		return f.created, nil
	}
	u := *user
	u.ID = "user-1"
	return &u, nil
}

func TestUserService_CreateUser_PublishesUserRegistered(t *testing.T) {
	repo := &createUserFakeRepo{}
	kafkaGw := &fakeKafkaGateway{}
	svc := NewUserService(repo, kafkaGw)

	got, err := svc.CreateUser(context.Background(), serviceEntity.RegisterInput{
		Name:     "Jane Doe",
		Email:    "jane@example.com",
		Phone:    "+15550001111",
		Password: "correct-password",
		Role:     serviceEntity.RoleDriver,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "user-1" {
		t.Errorf("got user ID %q, want %q", got.ID, "user-1")
	}
	if len(kafkaGw.published) != 1 {
		t.Fatalf("got %d published events, want 1", len(kafkaGw.published))
	}
	if kafkaGw.published[0].ID != "user-1" {
		t.Errorf("published event user ID = %q, want %q", kafkaGw.published[0].ID, "user-1")
	}
}

func TestUserService_CreateUser_InvalidRole(t *testing.T) {
	repo := &createUserFakeRepo{}
	kafkaGw := &fakeKafkaGateway{}
	svc := NewUserService(repo, kafkaGw)

	_, err := svc.CreateUser(context.Background(), serviceEntity.RegisterInput{
		Name:     "Jane Doe",
		Email:    "jane@example.com",
		Password: "correct-password",
		Role:     serviceEntity.RoleAdmin,
	})

	if !errors.Is(err, errorsx.ErrInvalidRole) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidRole)
	}
	if len(kafkaGw.published) != 0 {
		t.Error("expected no event published for an invalid role")
	}
}

func TestUserService_CreateUser_PublishFailureDoesNotFailRegistration(t *testing.T) {
	repo := &createUserFakeRepo{}
	kafkaGw := &fakeKafkaGateway{err: errors.New("broker unreachable")}
	svc := NewUserService(repo, kafkaGw)

	got, err := svc.CreateUser(context.Background(), serviceEntity.RegisterInput{
		Name:     "Jane Doe",
		Email:    "jane@example.com",
		Password: "correct-password",
		Role:     serviceEntity.RoleUser,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "user-1" {
		t.Errorf("got user ID %q, want %q", got.ID, "user-1")
	}
}

func TestUserService_CreateUser_RepoError(t *testing.T) {
	repo := &createUserFakeRepo{err: errorsx.ErrUserAlreadyExists}
	kafkaGw := &fakeKafkaGateway{}
	svc := NewUserService(repo, kafkaGw)

	_, err := svc.CreateUser(context.Background(), serviceEntity.RegisterInput{
		Name:     "Jane Doe",
		Email:    "jane@example.com",
		Password: "correct-password",
		Role:     serviceEntity.RoleUser,
	})

	if !errors.Is(err, errorsx.ErrUserAlreadyExists) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserAlreadyExists)
	}
	if len(kafkaGw.published) != 0 {
		t.Error("expected no event published when repo insert fails")
	}
}

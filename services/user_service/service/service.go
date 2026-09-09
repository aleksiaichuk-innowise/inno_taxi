package service

import (
	"context"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *serviceEntity.User) (*serviceEntity.User, error)
	FindByLogin(ctx context.Context, login string) (*serviceEntity.User, error)
	FindByID(ctx context.Context, id string) (*serviceEntity.User, error)
	UpdateProfile(ctx context.Context, id, name, email, phone string) (*serviceEntity.User, error)
	SetPassword(ctx context.Context, id string, password string) (err error)
	DeleteProfile(ctx context.Context, id string) (err error)
	AddRole(ctx context.Context, id, role string) error
}

type KafkaGateway interface {
	PublishUserRegistered(ctx context.Context, user serviceEntity.User) error
}

type UserService struct {
	userRepo UserRepository
	kafkaGw  KafkaGateway
}

func NewUserService(userRepo UserRepository, kafkaGw KafkaGateway) *UserService {
	return &UserService{userRepo: userRepo, kafkaGw: kafkaGw}
}

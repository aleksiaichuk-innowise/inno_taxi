package service

import "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/repository/mongo"

type UserService struct {
	userRepo *mongo.UserRepository
}

func NewUserService(userRepo *mongo.UserRepository) *UserService {
	return &UserService{userRepo}
}

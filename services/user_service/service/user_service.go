package service

import (
	"fmt"
	"strings"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/repository/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/consts"
	"github.com/gin-gonic/gin"
)

type UserService struct {
	userRepo *mongo.UserRepository
}

func (s UserService) CreateUser(c *gin.Context, input service.RegisterInput) error {
	role := strings.ToLower(string(input.Role))
	if role != consts.UserRole || role != consts.DriverRole {
		return fmt.Errorf("invalid role")
	}

	s.userRepo.CreateUser()

	return nil
}

func NewUserService(userRepo *mongo.UserRepository) *UserService {
	return &UserService{userRepo}
}

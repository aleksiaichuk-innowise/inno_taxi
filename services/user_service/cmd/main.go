package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/app/db/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/config"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/handler/http"
	mongo_repo "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/repository/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// connections
	mongoConn, err := mongo.New(ctx, cfg.Mongo)
	if err != nil {
		log.Fatal(err)
	}

	// repositories
	userRepo := mongo_repo.NewUserRepository(mongoConn)


	// services
	userSrv := service.NewUserService(userRepo)

	// deps
	validate := validator.New()

	// handlers

	h := http.NewHttpHandler(userSrv, validate)
	r := gin.Default()

	r.POST("/register", h.Register)
	r.POST("/login", h.Login)

	r.Run(fmt.Sprintf("%s:%s", cfg.Host.Host, cfg.Host.Port))

}

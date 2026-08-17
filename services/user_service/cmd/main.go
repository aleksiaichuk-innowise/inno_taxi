package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/app/db/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/config"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/handler/http"
	mongo_migration "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/migrations/mongo"
	mongo_repo "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/repository/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/validation"
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
	defer func(mongoConn *mongo.MongoClient, ctx context.Context) {
		_ = mongoConn.Close(ctx)
	}(mongoConn, ctx)

	if err := mongo_migration.RunMigrations(ctx, *mongoConn); err != nil {
		slog.Error("Failed to run migrations", "error", err)
	}

	// repositories
	userRepo := mongo_repo.NewUserRepository(mongoConn)

	// services
	userSrv := service.NewUserService(userRepo)

	// deps
	validate := validator.New()
	if err := validation.Register(validate); err != nil {
		log.Fatal(err)
	}

	// handlers

	h := http.NewHttpHandler(userSrv, validate)
	r := gin.Default()

	r.POST("/register", h.Register)
	r.POST("/internal/verify-credentials", h.VerifyCredentials)

	if err := r.Run(fmt.Sprintf("%s:%s", cfg.Host.Host, cfg.Host.Port)); err != nil {
		slog.Error("Failed to run server", "error", err)
	}

}

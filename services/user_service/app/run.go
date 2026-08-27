package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	dbmongo "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/app/db/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/config"
	entity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	httphandler "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/handler/http"
	mongomigration "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/migrations/mongo"
	mongorepo "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/repository/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/validation"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/transport/http/middleware"
)

func Run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mongoConn, err := dbmongo.New(ctx, cfg.Mongo)
	if err != nil {
		return fmt.Errorf("connect mongo: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoConn.Close(closeCtx); err != nil {
			slog.Error("close mongo connection", "error", err)
		}
	}()

	if err := mongomigration.RunMigrations(ctx, *mongoConn); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	userRepo := mongorepo.NewUserRepository(mongoConn)
	userSrv := service.NewUserService(userRepo)

	validate := validator.New()
	if err := validation.Register(validate); err != nil {
		return fmt.Errorf("register validators: %w", err)
	}

	h := httphandler.NewHttpHandler(userSrv, validate)
	r := gin.Default()
	registerRoutes(r, h, cfg)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Host.Host, cfg.Host.Port),
		Handler: r,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}

func registerRoutes(r *gin.Engine, h *httphandler.Handler, cfg *config.Config) {
	r.POST("/register", h.Register)
	r.POST("/internal/verify-credentials", h.VerifyCredentials)

	profile := r.Group("/profile")
	profile.Use(middleware.Auth(cfg.JWT.Secret))
	{
		profile.GET("", h.Profile)
		profile.PATCH("", h.UpdateProfile)
		profile.DELETE("", h.DeleteProfile)
		profile.POST("/password", h.UpdatePassword)
	}

	admin := r.Group("/admin")
	admin.Use(middleware.Auth(cfg.JWT.Secret))
	{
		admin.POST("/users/:id/analyst-role", middleware.RequireRole(string(entity.RoleAdmin)), h.AssignRoleAnalytic)
	}
}

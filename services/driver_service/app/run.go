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

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app/db/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/config"
	http_handler "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/handler/http"
	mongo_migration "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/migrations/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/repository"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func Run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	mongoConn, err := mongo.New(ctx, cfg.Mongo)

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

	if err := mongo_migration.RunMigrations(ctx, *mongoConn); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	repo := repository.NewDriverRepository(*mongoConn)
	s := service.NewDriverService(repo)

	router := gin.Default()

	v := validator.New()
	h := http_handler.NewDriverHandler(s, v)

	registerRoutes(router, h)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Host.Host, cfg.Host.Port),
		Handler: router,
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
	return nil
}

func registerRoutes(r *gin.Engine, h *http_handler.Handler) {
	r.POST("/internal/drivers", h.Register)
}

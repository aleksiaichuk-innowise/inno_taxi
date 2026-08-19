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
	"github.com/gin-gonic/gin"
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

	router := gin.Default()

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

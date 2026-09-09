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

	redisdb "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/app/db/redis"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/config"
	userservicegw "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/gateway/user_service"
	http_handler "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/handler/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/repository/redis_repo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/service"
	"github.com/gin-gonic/gin"
)

func Run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	redisClient, err := redisdb.New(ctx, cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error("close redis connection", "error", err)
		}
	}()

	sessions := redis_repo.NewSessionRepository(redisClient)
	userServiceGateway := userservicegw.NewUserServiceGateway(cfg.UserService.BaseURL, nil)
	authSvc := service.NewAuthService(sessions, userServiceGateway, cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	h := http_handler.NewHandler(authSvc)
	r := gin.Default()
	registerRoutes(r, h)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Host.Host, cfg.Host.Port),
		Handler: r,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info(fmt.Sprintf("auth service listening on %s", srv.Addr))
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

	slog.Info("auth service stopped gracefully")
	return nil
}

func registerRoutes(r *gin.Engine, h *http_handler.Handler) {
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/logout", h.Logout)
	// GET, not POST: nginx's auth_request module always issues its
	// subrequest as GET regardless of the original request's method, and
	// Validate has no side effects anyway (Redis reads only).
	r.GET("/validate", h.Validate)
}

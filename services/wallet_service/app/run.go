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

	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/app/db/postgres"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/app/db/redis"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/config"
	http_handler "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/handler/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/repository/pg_repo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/repository/redis_repo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/service"
	"github.com/gin-gonic/gin"
)

func Run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := postgres.NewDB(ctx, cfg.DbConn)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer db.Close()

	redisClient, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error("close redis connection", "error", err)
		}
	}()

	repo := pg_repo.NewPgRepo(db)
	cache := redis_repo.NewTransactionCache(redisClient)
	walletSvc := service.NewWalletService(repo, cache)

	h := http_handler.NewHandler(walletSvc)
	r := gin.Default()
	registerRoutes(r, h)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.HttpHost.Host, cfg.HttpHost.Port),
		Handler: r,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info(fmt.Sprintf("wallet service listening on %s", srv.Addr))
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

	slog.Info("wallet service stopped gracefully")
	return nil
}

func registerRoutes(r *gin.Engine, h *http_handler.Handler) {
	wallets := r.Group("/internal/wallets")
	wallets.POST("", h.CreateWallet)
	wallets.GET("/:user_id", h.GetWallet)
	wallets.POST("/:user_id/charge", h.Charge)
	wallets.POST("/:user_id/refund", h.Refund)
	wallets.GET("/:user_id/transactions", h.ListTransactions)
}

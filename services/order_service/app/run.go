package app

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/app/db/elastic"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/app/db/postgres"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/config"
)

func Run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbConn, err := postgres.NewPgPool(ctx, cfg.DbConn)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	es, err := elastic.NewESClient(ctx, cfg.EsConn)
	if err != nil {
		return err
	}
	defer es.Close(ctx)

	slog.Info("order service starting")

	<-ctx.Done()
	slog.Info("shutdown signal received, order service stopped")

	return nil
}

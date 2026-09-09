package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/app/db/clickhouse"
	appkafka "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/app/kafka"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/config"
	http_handler "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/handler/http"
	kafkahandler "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/handler/kafka"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/repository/ch_repo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/service"
	"github.com/gofiber/fiber/v2"
)

func Run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	chConn, err := clickhouse.New(ctx, cfg.ClickHouse)
	if err != nil {
		return fmt.Errorf("connect clickhouse: %w", err)
	}
	defer chConn.Close()

	repo := ch_repo.NewClickHouseRepository(chConn)
	analyticSvc := service.NewAnalyticService(repo)

	consumerGroup, err := appkafka.NewConsumerGroup(cfg.Kafka.Brokers)
	if err != nil {
		return fmt.Errorf("new kafka consumer group: %w", err)
	}
	defer func() {
		if err := consumerGroup.Close(); err != nil {
			slog.Error("close kafka consumer group", "error", err)
		}
	}()

	consumer := kafkahandler.NewOrderCreatedConsumer(analyticSvc)
	go func() {
		for ctx.Err() == nil {
			if err := consumerGroup.Consume(ctx, []string{kafkahandler.TopicOrderCreated}, consumer); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("kafka consumer group session ended", "error", err)
			}
		}
	}()
	go func() {
		for err := range consumerGroup.Errors() {
			slog.Error("kafka consumer group error", "error", err)
		}
	}()

	h := http_handler.NewHandler(analyticSvc)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	registerRoutes(app, h)

	serverErr := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.HttpHost.Host, cfg.HttpHost.Port)
		slog.Info(fmt.Sprintf("analytic service listening on %s", addr))
		if err := app.Listen(addr); err != nil {
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
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	slog.Info("analytic service stopped gracefully")
	return nil
}

func registerRoutes(app *fiber.App, h *http_handler.Handler) {
	orders := app.Group("/orders")
	orders.Get("/stats", h.GetOrderStats)
	orders.Get("/daily", h.GetDailyOrderCounts)
}

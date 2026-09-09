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

	orderConsumerGroup, err := appkafka.NewConsumerGroup(cfg.Kafka.Brokers, appkafka.ConsumerGroupIDOrderCreated)
	if err != nil {
		return fmt.Errorf("new order_created kafka consumer group: %w", err)
	}
	defer func() {
		if err := orderConsumerGroup.Close(); err != nil {
			slog.Error("close order_created kafka consumer group", "error", err)
		}
	}()

	orderConsumer := kafkahandler.NewOrderCreatedConsumer(analyticSvc)
	go func() {
		for ctx.Err() == nil {
			if err := orderConsumerGroup.Consume(ctx, []string{kafkahandler.TopicOrderCreated}, orderConsumer); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("order_created kafka consumer group session ended", "error", err)
			}
		}
	}()
	go func() {
		for err := range orderConsumerGroup.Errors() {
			slog.Error("order_created kafka consumer group error", "error", err)
		}
	}()

	userConsumerGroup, err := appkafka.NewConsumerGroup(cfg.Kafka.Brokers, appkafka.ConsumerGroupIDUserRegistered)
	if err != nil {
		return fmt.Errorf("new user_registered kafka consumer group: %w", err)
	}
	defer func() {
		if err := userConsumerGroup.Close(); err != nil {
			slog.Error("close user_registered kafka consumer group", "error", err)
		}
	}()

	userConsumer := kafkahandler.NewUserRegisteredConsumer(analyticSvc)
	go func() {
		for ctx.Err() == nil {
			if err := userConsumerGroup.Consume(ctx, []string{kafkahandler.TopicUserRegistered}, userConsumer); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("user_registered kafka consumer group session ended", "error", err)
			}
		}
	}()
	go func() {
		for err := range userConsumerGroup.Errors() {
			slog.Error("user_registered kafka consumer group error", "error", err)
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

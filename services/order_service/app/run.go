package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/app/db/elastic"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/app/db/postgres"
	app_kafka "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/app/kafka"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/config"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/gateway"
	driver_gateway "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/gateway/driver_service"
	wallet_gateway "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/gateway/wallet_service"
	grpc_srv "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/handler/grpc"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/repository/pg_repo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/transport/grpc/interceptor"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	// -- kafka
	kafkaProducer, err := app_kafka.NewPublisher(cfg.KafkaConf.Brokers)
	if err != nil {
		slog.Error("order service failed to init kafka producer", "error", err)
		return err
	}
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			slog.Warn("order service failed to close kafka producer", "error", err)
		}
	}()
	kafkaGateway := gateway.NewKafkaGateway(kafkaProducer)
	walletGateway := wallet_gateway.NewWalletGateway(cfg.WalletService.BaseURL, nil)
	driverGateway := driver_gateway.NewDriverGateway(cfg.DriverService.BaseURL, nil)

	// -- repo
	repo := pg_repo.NewPgRepo(dbConn)

	// services
	orderService := service.NewOrderService(repo, kafkaGateway, walletGateway, driverGateway, nil)

	// -- Grpc

	grpcListener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.Grpc.Host, cfg.Grpc.Port))
	if err != nil {
		slog.Error("order service failed to listen", "error", err)
		return err
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(
		interceptor.AuthInterceptor(cfg.JWT.Secret),
	))
	order_service.RegisterOrderServiceServer(grpcServer, grpc_srv.NewOrderServer(orderService))

	go func() {
		slog.Info(fmt.Sprintf("grpc server listening on port %s", cfg.Grpc.Port))
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			slog.Error("order service failed to serve", "error", err)
		}
	}()

	// -- GRPC-gateway
	gw := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = order_service.RegisterOrderServiceHandlerFromEndpoint(ctx, gw, grpcListener.Addr().String(), opts)
	if err != nil {
		slog.Error("order service failed to register gateway", "error", err)
		return err
	}

	httpAddr := fmt.Sprintf("%s:%s", cfg.HttpHost.Host, cfg.HttpHost.Port)
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: gw,
	}

	go func() {
		slog.Info(fmt.Sprintf("http gateway listening on %s", httpAddr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http gateway failed to serve", "error", err)
		}
	}()

	// -- Graceful shutdown
	<-ctx.Done()
	slog.Info("shutdown signal received, stopping order service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Warn("http gateway shutdown error", "error", err)
	}

	select {
	case <-stopped:
		slog.Info("gRPC server stopped gracefully")
	case <-shutdownCtx.Done():
		slog.Warn("gRPC server forced to stop due to timeout")
		grpcServer.Stop()
	}

	slog.Info("order service stopped successfully")
	return nil
}

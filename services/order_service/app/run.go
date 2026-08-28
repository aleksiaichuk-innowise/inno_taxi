package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/app/db/elastic"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/app/db/postgres"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/config"
	grpc_srv "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/handler/grpc"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/transport/grpc/interceptor"
	"google.golang.org/grpc"
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

	// -- Grpc

	grpcListener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.Grpc.Host, cfg.Grpc.Port))
	if err != nil {
		slog.Error("order service failed to listen", "error", err)
		return err
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(
		interceptor.AuthInterceptor(cfg.JWT.Secret),
	))
	order_service.RegisterOrderServiceServer(grpcServer, grpc_srv.NewOrderServer())

	go func() {
		slog.Info(fmt.Sprintf("grpc server listening on port %s", cfg.Grpc.Port))
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			slog.Error("order service failed to serve", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, stopping order service...")

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

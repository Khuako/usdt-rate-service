package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	ratespb "github.com/Khuako/usdt-rate-service/gen/rates"
	"github.com/Khuako/usdt-rate-service/internal/config"
	"github.com/Khuako/usdt-rate-service/internal/handler/grpchandler"
	"github.com/Khuako/usdt-rate-service/internal/kraken"
	"github.com/Khuako/usdt-rate-service/internal/outbox"
	outboxrepo "github.com/Khuako/usdt-rate-service/internal/outbox/repository"
	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/Khuako/usdt-rate-service/internal/rates/publisher"
	"github.com/Khuako/usdt-rate-service/internal/rates/repository"
	"github.com/Khuako/usdt-rate-service/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync()
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, unix.SIGTERM)
	defer stop()
	err = run(ctx, logger)
	if err != nil {
		logger.Error("server failed", zap.Error(err))
		_ = logger.Sync()
		os.Exit(1)
	}
}
func run(ctx context.Context, logger *zap.Logger) error {

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("%w: config loading error", err)
	}
	provider, err := telemetry.NewTracerProvider(ctx)
	if err != nil {
		return fmt.Errorf("%w: error creating tracer provider", err)
	}
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := provider.Shutdown(shutdownCtx); err != nil {
			logger.Warn("tracing shutdown failed", zap.Error(err))
		}
	}()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("%w: database connecting error", err)
	}
	defer pool.Close()
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = pool.Ping(pingCtx)
	if err != nil {
		return fmt.Errorf("%w: database ping error", err)
	}
	repo := repository.NewRepository(pool)
	client := kraken.NewClient(cfg.KrakenBaseURL)

	service := rates.NewService(repo, client)
	handler := grpchandler.NewHandler(service, logger)
	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	defer listener.Close()
	if err != nil {
		return fmt.Errorf("%w: grpc listener error", err)
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(grpchandler.LoggingInterceptor(logger)), grpc.StatsHandler(otelgrpc.NewServerHandler()))
	ratespb.RegisterRatesServiceServer(server, handler)
	errChan := make(chan error, 1)
	healthHandler := grpchandler.NewHealthHandler(pool)
	healthpb.RegisterHealthServer(server, healthHandler)

	outboxCtx, stopOutbox := context.WithCancel(ctx)
	defer stopOutbox()

	var outboxStopped chan struct{}
	if cfg.KafkaBrokers != "" {
		var outboxService *outbox.Service
		kafkaClient, err := kgo.NewClient(
			kgo.SeedBrokers(strings.Split(cfg.KafkaBrokers, ",")...),
		)
		if err != nil {
			logger.Error("kafka client failed", zap.String("error", err.Error()))
			return err
		}
		defer kafkaClient.Close()
		outboxService = outbox.NewService(
			outboxrepo.New(pool),
			publisher.New(kafkaClient),
		)
		outboxStopped = make(chan struct{})

		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			defer close(outboxStopped)
			for {
				select {
				case <-outboxCtx.Done():
					return
				case <-ticker.C:
					pContext, stopPublish := context.WithTimeout(outboxCtx, time.Second*5)
					publishErr := outboxService.PublishPending(pContext)
					stopPublish()
					if outboxCtx.Err() != nil {
						return
					}
					if publishErr != nil {
						logger.Error("error publish", zap.String("error", publishErr.Error()))
					}
				}
			}
		}()
	}

	go func() {
		errChan <- server.Serve(listener)
	}()
	logger.Info("server has started", zap.String("address:", cfg.GRPCAddr))
	select {
	case err = <-errChan:
	case <-ctx.Done():
	}
	logger.Info("server is stopping")
	healthHandler.Shutdown()
	serverStopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(serverStopped)
	}()
	t := time.NewTimer(time.Second * 5)
	defer t.Stop()
	select {
	case <-serverStopped:
	case <-t.C:
		logger.Warn("forcing stop")
		server.Stop()
	}
	if outboxStopped != nil {
		stopOutbox()
		<-outboxStopped
	}
	logger.Info("server has stopped")
	return err
}

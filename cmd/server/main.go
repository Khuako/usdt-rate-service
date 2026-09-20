package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"time"

	ratespb "github.com/Khuako/usdt-rate-service/gen/rates"
	"github.com/Khuako/usdt-rate-service/internal/config"
	"github.com/Khuako/usdt-rate-service/internal/handler/grpchandler"
	"github.com/Khuako/usdt-rate-service/internal/kraken"
	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/Khuako/usdt-rate-service/internal/rates/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, unix.SIGTERM)
	defer stop()
	err := run(ctx)
	if err != nil {
		os.Exit(1)
	}
}
func run(ctx context.Context) error {

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = pool.Ping(pingCtx)
	if err != nil {
		return err
	}
	repo := repository.NewRepository(pool)
	client := kraken.NewClient(cfg.KrakenBaseURL)

	service := rates.NewService(repo, client)
	handler := grpchandler.NewHandler(service)
	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	ratespb.RegisterRatesServiceServer(server, handler)
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Serve(listener)
	}()
	select {
	case err = <-errChan:
	case <-ctx.Done():
	}
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
		server.Stop()
	}
	return err
}

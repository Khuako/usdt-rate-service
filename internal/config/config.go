package config

import (
	"errors"
	"flag"
	"io"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL   string
	GRPCAddr      string
	KrakenBaseURL string
}

var (
	ErrEmptyGRPCAddr      = errors.New("GRPC address is empty")
	ErrEmptyDB            = errors.New("database URL is empty")
	ErrEmptyKrakenBaseUrl = errors.New("kraken base url is empty")
)

func Load(args []string) (Config, error) {
	cfg := Config{
		GRPCAddr:      ":50051",
		KrakenBaseURL: "https://api.kraken.com",
	}
	if value, exists := os.LookupEnv("GRPC_ADDR"); exists {
		cfg.GRPCAddr = value
	}
	if value, exists := os.LookupEnv("DATABASE_URL"); exists {
		cfg.DatabaseURL = value
	}
	if value, exists := os.LookupEnv("KRAKEN_BASE_URL"); exists {
		cfg.KrakenBaseURL = value
	}
	fs := flag.NewFlagSet("app", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", cfg.GRPCAddr, "gRPC listen address")
	fs.StringVar(&cfg.DatabaseURL, "database-url", cfg.DatabaseURL, "database-url")
	fs.StringVar(&cfg.KrakenBaseURL, "kraken-base-url", cfg.KrakenBaseURL, "base url")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(cfg.GRPCAddr) == "" {
		return Config{}, ErrEmptyGRPCAddr
	}
	if strings.TrimSpace(cfg.KrakenBaseURL) == "" {
		return Config{}, ErrEmptyKrakenBaseUrl
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return Config{}, ErrEmptyDB
	}
	return cfg, nil
}

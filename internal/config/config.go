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
	KafkaBrokers  string
}

var (
	ErrEmptyGRPCAddr       = errors.New("GRPC address is empty")
	ErrEmptyDB             = errors.New("database URL is empty")
	ErrEmptyKrakenBaseUrl  = errors.New("kraken base url is empty")
	ErrInvalidKafkaBrokers = errors.New("kafka brokers contains an empty address")
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
	if value, exists := os.LookupEnv("KAFKA_BROKERS"); exists {
		cfg.KafkaBrokers = value
	}
	fs := flag.NewFlagSet("app", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", cfg.GRPCAddr, "gRPC listen address")
	fs.StringVar(&cfg.DatabaseURL, "database-url", cfg.DatabaseURL, "database-url")
	fs.StringVar(&cfg.KrakenBaseURL, "kraken-base-url", cfg.KrakenBaseURL, "base url")
	fs.StringVar(&cfg.KafkaBrokers, "kafka-brokers", cfg.KafkaBrokers, "kafka brokers")
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
	cfg.KafkaBrokers = strings.TrimSpace(cfg.KafkaBrokers)
	if cfg.KafkaBrokers != "" {
		brokers := strings.Split(cfg.KafkaBrokers, ",")
		for i, broker := range brokers {
			brokers[i] = strings.TrimSpace(broker)
			if brokers[i] == "" {
				return Config{}, ErrInvalidKafkaBrokers
			}
		}
		cfg.KafkaBrokers = strings.Join(brokers, ",")
	}
	return cfg, nil
}

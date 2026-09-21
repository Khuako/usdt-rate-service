package testutils

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/jackc/pgx/v5/pgxpool"
)

func AssertRateEqual(t *testing.T, got, want rates.Rate) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("ID: got %v, want %v", got.ID, want.ID)
	}
	if !got.Ask.Equal(want.Ask) {
		t.Errorf("Ask: got %v, want %v", got.Ask, want.Ask)
	}
	if !got.Bid.Equal(want.Bid) {
		t.Errorf("Bid: got %v, want %v", got.Bid, want.Bid)
	}
	if !got.ReceivedAt.Equal(want.ReceivedAt) {
		t.Errorf("ReceivedAt: got %v, want %v", got.ReceivedAt, want.ReceivedAt)
	}
	if got.Calculation != want.Calculation {
		t.Errorf("Calculation: got %+v, want %+v", got.Calculation, want.Calculation)
	}
}
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	return pool
}

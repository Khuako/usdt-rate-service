package repository

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/Khuako/usdt-rate-service/internal/testutils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
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
func TestRepository_Save(t *testing.T) {
	pool := setupTestDB(t)
	tests := []rates.Calculation{
		{Method: rates.MethodTopN, N: 2},
		{Method: rates.MethodAvgNM, N: 1, M: 3},
	}
	for _, calc := range tests {
		t.Run(string(calc.Method), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			repo := NewRepository(pool)
			want := rates.Rate{
				ID:          uuid.New(),
				Ask:         decimal.RequireFromString("0.9992"),
				Bid:         decimal.RequireFromString("0.9991"),
				ReceivedAt:  time.Date(2026, time.September, 19, 12, 0, 0, 123456000, time.UTC),
				Calculation: calc,
			}
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if _, err := pool.Exec(ctx, `delete from outbox_events where message_key = $1`, want.ID.String()); err != nil {
					t.Errorf("delete test outbox events: %v", err)
				}
				if _, err := pool.Exec(ctx, `delete from rates where id = $1`, want.ID); err != nil {
					t.Errorf("delete test rate: %v", err)
				}
			})
			if err := repo.Save(ctx, want); err != nil {
				t.Fatalf("save: %v", err)
			}

			var got rates.Rate
			var m *int
			err := pool.QueryRow(ctx, `select * from rates where id = $1`, want.ID).Scan(
				&got.ID, &got.Ask, &got.Bid, &got.ReceivedAt, &got.Method, &got.N, &m,
			)
			if err != nil {
				t.Fatalf("read saved rate: %v", err)
			}
			if calc.Method == rates.MethodTopN {
				if m != nil {
					t.Fatalf("topN m: got %d, want NULL", *m)
				}
			} else {
				if m == nil {
					t.Fatal("avgNM m: got NULL, want a value")
				}
				got.M = *m
			}
			testutils.AssertRateEqual(t, got, want)
			assertSavedEvent(t, ctx, pool, want)
		})
	}
}

func assertSavedEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want rates.Rate) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `select count(*) from outbox_events where message_key = $1`, want.ID.String()).Scan(&count); err != nil {
		t.Fatalf("count outbox events: %v", err)
	}
	if count != 1 {
		t.Fatalf("outbox events: got %d, want 1", count)
	}

	var id uuid.UUID
	var topic string
	var payload []byte
	var createdAt time.Time
	var publishedAt *time.Time
	err := pool.QueryRow(ctx,
		`select id, topic, payload, created_at, published_at from outbox_events where message_key = $1`,
		want.ID.String(),
	).Scan(&id, &topic, &payload, &createdAt, &publishedAt)
	if err != nil {
		t.Fatalf("read outbox event: %v", err)
	}
	if topic != "rates.calculated" {
		t.Errorf("topic: got %q, want rates.calculated", topic)
	}
	if publishedAt != nil {
		t.Errorf("published_at: got %v, want NULL", publishedAt)
	}
	if createdAt.IsZero() {
		t.Error("created_at is zero")
	}

	var event rates.RateCalculated
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if event.EventID == uuid.Nil || event.EventID != id {
		t.Errorf("event_id: got %v, want nonzero outbox ID %v", event.EventID, id)
	}
	if event.Type != "rate.calculated" || event.Version != 1 || event.Pair != "USDT/USD" {
		t.Errorf("event metadata: got type=%q version=%d pair=%q", event.Type, event.Version, event.Pair)
	}
	if event.RateID != want.ID {
		t.Errorf("rate_id: got %v, want %v", event.RateID, want.ID)
	}
	if event.Ask != want.Ask.String() || event.Bid != want.Bid.String() {
		t.Errorf("event prices: got ask=%q bid=%q, want ask=%q bid=%q", event.Ask, event.Bid, want.Ask.String(), want.Bid.String())
	}
	if !event.ReceivedAt.Equal(want.ReceivedAt) {
		t.Errorf("received_at: got %v, want %v", event.ReceivedAt, want.ReceivedAt)
	}
	if event.OccurredAt.IsZero() {
		t.Error("occurred_at is zero")
	}
	if event.Method != want.Method || event.N != want.N || event.M != want.M {
		t.Errorf("event calculation: got method=%q n=%d m=%d, want %+v", event.Method, event.N, event.M, want.Calculation)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatalf("decode event fields: %v", err)
	}
	var m *int
	if err := json.Unmarshal(fields["m"], &m); err != nil || m == nil {
		t.Errorf("event must contain numeric m got %s (error: %v)", fields["m"], err)
	}
}

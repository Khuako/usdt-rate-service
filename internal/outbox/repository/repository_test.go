package repository

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/Khuako/usdt-rate-service/internal/outbox"
	"github.com/Khuako/usdt-rate-service/internal/testutils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepository_ListPending(t *testing.T) {
	pool := testutils.SetupTestDB(t)
	t.Run("repository logic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		publishedTime := time.Now().Add(time.Hour)
		messages := []outbox.Message{{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Topic:       "topic1",
			MessageKey:  "message1",
			Payload:     []byte(`[1, 2, 3, 4, 5]`),
			CreatedAt:   time.Now().Truncate(time.Microsecond),
			PublishedAt: nil,
		}, {
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Topic:       "topic2",
			MessageKey:  "message2",
			Payload:     []byte(`[1, 2, 3, 4, 5,6]`),
			CreatedAt:   time.Now().Add(time.Second * 10).Truncate(time.Microsecond),
			PublishedAt: nil,
		}, {
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Topic:       "topic3",
			MessageKey:  "message3",
			Payload:     []byte(`[1, 2, 3, 4, 5,7,8]`),
			CreatedAt:   time.Now().Add(time.Second * 11).Truncate(time.Microsecond),
			PublishedAt: &publishedTime,
		}}
		schemaName := "test_outbox_" + uuid.New().String()
		schema := pgx.Identifier{schemaName}.Sanitize()

		if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
			t.Fatalf("create schema: %v", err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
				t.Errorf("drop test schema: %v", err)
			}
		})
		config := pool.Config()
		config.ConnConfig.RuntimeParams["search_path"] = schema
		testPool, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			t.Fatalf("create pool: %v", err)
		}
		t.Cleanup(testPool.Close)
		if _, err := testPool.Exec(ctx, `
				CREATE TABLE outbox_events
				(LIKE public.outbox_events INCLUDING ALL)
			`); err != nil {
			t.Fatalf("creating test table: %v", err)
		}
		for _, v := range messages {
			if _, err := testPool.Exec(
				ctx,
				`insert into outbox_events values ($1,$2,$3,$4,$5,$6)`,
				v.ID,
				v.Topic,
				v.MessageKey,
				v.Payload,
				v.CreatedAt,
				v.PublishedAt,
			); err != nil {
				t.Fatalf("saving message: %v", err)
			}
		}
		repo := New(testPool)
		singleValue, err := repo.ListPending(ctx, 1)
		if err != nil {
			t.Fatalf("list pending single: %v", err)
		}
		if len(singleValue) != 1 {
			t.Fatalf("wrong limit, want 1 element, got: %v", singleValue)
		}
		values, err := repo.ListPending(ctx, 100)
		if err != nil {
			t.Fatalf("list pending : %v", err)
		}
		if len(values) != 2 {
			t.Fatalf("wrong list length, want: 2, got: %d", len(values))
		}
		for i := range values {
			if values[i].ID != messages[i].ID {
				t.Errorf("wrong id at %d, want: %v, got: %v", i, messages[i].ID, values[i].ID)
			}
			if values[i].PublishedAt != nil {
				t.Errorf("PublishedAt: got %v, want nil", values[i].PublishedAt)
			}
			if !values[i].CreatedAt.Equal(messages[i].CreatedAt) {
				t.Errorf("wrong CreatedAt at %d, want: %v, got: %v", i, messages[i].CreatedAt, values[i].CreatedAt)
			}
			var gotPayload, wantPayload []int

			if err := json.Unmarshal(values[i].Payload, &gotPayload); err != nil {
				t.Fatalf("decode got message fields: %v", err)
			}
			if err := json.Unmarshal(messages[i].Payload, &wantPayload); err != nil {
				t.Fatalf("decode want message fields: %v", err)
			}
			if !slices.Equal(gotPayload, wantPayload) {
				t.Errorf("wrong payload, got: %v, want :%v", gotPayload, wantPayload)
			}
			if values[i].MessageKey != messages[i].MessageKey {
				t.Errorf("wrong MessageKey at %d, want: %v, got: %v", i, messages[i].MessageKey, values[i].MessageKey)
			}
			if values[i].Topic != messages[i].Topic {
				t.Errorf("wrong Topic at %d, want: %v, got: %v", i, messages[i].Topic, values[i].Topic)
			}
		}
		err = repo.MarkPublished(ctx, values[1].ID)
		if err != nil {
			t.Fatalf("mark published: %v", err)
		}
		var publishedDate *time.Time
		err = testPool.QueryRow(ctx, `select published_at from outbox_events where id = $1`, values[1].ID).Scan(&publishedDate)
		if err != nil {
			t.Fatalf("getting published date: %v", err)
		}
		if publishedDate == nil {
			t.Fatalf("published date still nil, expected date")
		}
		values, err = repo.ListPending(ctx, 100)
		if err != nil {
			t.Fatalf("list pending after publication: %v", err)
		}
		if len(values) != 1 {
			t.Fatalf("pending after publication: got %d, want 1", len(values))
		}
		if values[0].ID != messages[0].ID {
			t.Errorf("remaining ID: got %v, want %v", values[0].ID, messages[0].ID)
		}
		if values[0].PublishedAt != nil {
			t.Errorf("remaining PublishedAt: got %v, want nil", values[0].PublishedAt)
		}
	})
}

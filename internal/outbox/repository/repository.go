package repository

import (
	"context"

	"github.com/Khuako/usdt-rate-service/internal/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db}
}

func (r *Repository) ListPending(ctx context.Context, limit int) ([]outbox.Message, error) {
	var events []outbox.Message
	rows, err := r.db.Query(
		ctx,
		`select * from outbox_events where published_at is null order by id limit $1`,
		limit,
	)
	if err != nil {
		return events, err
	}
	defer rows.Close()
	for rows.Next() {
		var e outbox.Message
		err = rows.Scan(&e.ID, &e.Topic, &e.MessageKey, &e.Payload, &e.CreatedAt, &e.PublishedAt)
		if err != nil {
			return events, err
		}
		events = append(events, e)
	}
	if rows.Err() != nil {
		return events, rows.Err()
	}
	return events, nil
}
func (r *Repository) MarkPublished(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `update outbox_events set published_at = now() where id = $1`, id)
	if err != nil {
		return err
	}
	return nil
}

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db}
}

func (r *Repository) Save(ctx context.Context, rate rates.Rate) error {
	ctx, span := otel.Tracer("usdt-rate-service/internal/rates/repository").Start(ctx, "repository.Save")
	defer span.End()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var m any
	if rate.Method == rates.MethodAvgNM {
		m = rate.M
	}
	_, err = tx.Exec(
		ctx,
		`insert into rates (id, ask, bid, received_at, method, n, m) values ($1, $2, $3, $4, $5, $6, $7)`,
		rate.ID,
		rate.Ask,
		rate.Bid,
		rate.ReceivedAt,
		rate.Method,
		rate.N,
		m,
	)
	if err != nil {
		var pgxErr *pgconn.PgError
		if errors.As(err, &pgxErr) {
			if pgxErr.Code == "23505" {
				span.RecordError(rates.ErrRateAlreadyExists)
				span.SetStatus(codes.Error, rates.ErrRateAlreadyExists.Error())
				return rates.ErrRateAlreadyExists
			}

		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "error saving rate")
		return fmt.Errorf("error saving rate: %w", err)
	}
	e := rates.RateCalculated{
		EventID:    uuid.New(),
		Type:       "rate.calculated",
		Version:    1,
		RateID:     rate.ID,
		Pair:       "USDT/USD",
		OccurredAt: time.Now(),
		Ask:        rate.Ask.String(),
		Bid:        rate.Bid.String(),
		ReceivedAt: rate.ReceivedAt,
		Method:     rate.Method,
		N:          rate.N,
		M:          rate.M,
	}
	payload, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		ctx,
		`insert into outbox_events values ($1, $2, $3, $4, $5)`,
		e.EventID,
		"rates.calculated",
		e.RateID.String(),
		payload,
		e.OccurredAt,
	)
	if err != nil {
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}
	return nil
}

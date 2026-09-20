package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Khuako/usdt-rate-service/internal/rates"
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
	var m any
	if rate.Method == rates.MethodAvgNM {
		m = rate.M
	}
	_, err := r.db.Exec(
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
	return nil
}

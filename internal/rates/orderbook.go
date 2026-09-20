package rates

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderBook struct {
	Asks       []decimal.Decimal `json:"asks"`
	Bids       []decimal.Decimal `json:"bids"`
	ReceivedAt time.Time
}

var (
	ErrRateAlreadyExists = errors.New("rate already exists")
)

type Rate struct {
	ID         uuid.UUID
	Ask        decimal.Decimal
	Bid        decimal.Decimal
	ReceivedAt time.Time
	Calculation
}
type RateCalculated struct {
	EventID    uuid.UUID `json:"event_id"`
	Type       string    `json:"type"`
	Version    int       `json:"version"`
	OccurredAt time.Time `json:"occurred_at"`
	RateID     uuid.UUID `json:"rate_id"`
	Pair       string    `json:"pair"`
	Ask        string    `json:"ask"`
	Bid        string    `json:"bid"`
	ReceivedAt time.Time `json:"received_at"`
	Method     Method    `json:"method"`
	N          int       `json:"n"`
	M          int       `json:"m"`
}

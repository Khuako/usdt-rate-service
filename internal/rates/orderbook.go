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

package rates

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type OrderBookProvider interface {
	GetOrderBook(ctx context.Context) (OrderBook, error)
}

type Repository interface {
	Save(ctx context.Context, rate Rate) error
}

type Service struct {
	repo     Repository
	provider OrderBookProvider
}

func NewService(repository Repository, provider OrderBookProvider) *Service {
	return &Service{
		repository,
		provider,
	}
}

func (s *Service) GetRates(ctx context.Context, calc Calculation) (Rate, error) {
	rate := Rate{
		Calculation: calc,
	}
	if calc.N < 1 {
		return Rate{}, ErrInvalidPos
	}
	if calc.Method == MethodAvgNM && calc.N > calc.M {
		return Rate{}, ErrInvalidRange
	}
	if calc.Method != MethodAvgNM && calc.Method != MethodTopN {
		return Rate{}, ErrWrongCalculationMethod
	}
	ob, err := s.provider.GetOrderBook(ctx)
	if err != nil {
		return Rate{}, fmt.Errorf("%w: error getting orderbook", err)
	}
	ask, nmErr := Calculate(ob.Asks, calc)
	if nmErr != nil {
		return Rate{}, fmt.Errorf("%w: ask calculation error", nmErr)
	}
	rate.Ask = ask
	bid, bidErr := Calculate(ob.Bids, calc)
	if bidErr != nil {
		return Rate{}, fmt.Errorf("%w: bid calculation error", bidErr)
	}
	rate.ID = uuid.New()
	rate.Bid = bid
	rate.ReceivedAt = ob.ReceivedAt
	err = s.repo.Save(ctx, rate)
	if err != nil {
		return Rate{}, fmt.Errorf("%w: saving error", err)
	}
	return rate, nil
}

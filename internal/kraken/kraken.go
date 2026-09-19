package kraken

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidResponse = errors.New("invalid response")
	ErrExchange        = errors.New("exchange returned an error")
	ErrEmptySide       = errors.New("order book side is empty")
	ErrInvalidPrice    = errors.New("invalid price")
	ErrEmptyResponse   = errors.New("the response is empty")
)

type depthResponse struct {
	Error  []string             `json:"error"`
	Result map[string]depthBook `json:"result"`
}
type depthBook struct {
	Asks [][]json.RawMessage `json:"asks"`
	Bids [][]json.RawMessage `json:"bids"`
}

func parseOrderBook(data []byte) (rates.OrderBook, error) {
	var orderBook rates.OrderBook
	var response depthResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return orderBook, ErrInvalidResponse
	}
	if len(response.Error) > 0 {
		return orderBook, fmt.Errorf("%w: error: %v", ErrExchange, response.Error[0])
	}
	book, ok := response.Result["USDTZUSD"]
	if !ok {
		return orderBook, ErrEmptyResponse
	}
	pricesAsks, err := parsePrices(book.Asks)
	if err != nil {
		return orderBook, fmt.Errorf("asks: %w", err)
	}
	pricesBids, err := parsePrices(book.Bids)
	if err != nil {
		return orderBook, fmt.Errorf("bids: %w", err)
	}
	orderBook.Asks = pricesAsks
	orderBook.Bids = pricesBids
	return orderBook, nil
}
func parsePrices(levels [][]json.RawMessage) ([]decimal.Decimal, error) {
	if len(levels) <= 0 {
		return []decimal.Decimal{}, ErrEmptySide
	}
	var prices []decimal.Decimal
	for i, val := range levels {
		if len(val) <= 0 {
			return prices, ErrInvalidPrice
		}
		var price string
		if err := json.Unmarshal(val[0], &price); err != nil {
			return prices, fmt.Errorf("%w: invalid price at: %d", ErrInvalidPrice, i+1)
		}
		a, err := decimal.NewFromString(price)
		if err != nil {
			return prices, fmt.Errorf("%w: invalid price at: %d", ErrInvalidPrice, i+1)
		}
		if a.IsNegative() || a.IsZero() {
			return prices, fmt.Errorf("%w: invalid price at: %d", ErrInvalidPrice, i+1)
		}
		prices = append(prices, a)
	}
	return prices, nil
}

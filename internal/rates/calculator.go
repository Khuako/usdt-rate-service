package rates

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

var (
	ErrEmptyPrices            = errors.New("prices are empty")
	ErrInvalidPos             = errors.New("position is beyond prices array")
	ErrInvalidRange           = errors.New("range is invalid")
	ErrWrongCalculationMethod = errors.New("wrong calculation method")
)

type Method string

const (
	MethodTopN  Method = "topN"
	MethodAvgNM Method = "avgNM"
)

type Calculation struct {
	Method Method
	N      int
	M      int
}

func Calculate(prices []decimal.Decimal, calc Calculation) (decimal.Decimal, error) {
	switch calc.Method {
	case MethodTopN:
		return topN(prices, calc.N)
	case MethodAvgNM:
		return avgNM(prices, calc.N, calc.M)
	default:
		return decimal.Decimal{}, fmt.Errorf("%w, calculation method: %v", ErrWrongCalculationMethod, calc.Method)
	}
}

func topN(prices []decimal.Decimal, n int) (decimal.Decimal, error) {
	var res decimal.Decimal
	if len(prices) <= 0 {
		return res, ErrEmptyPrices
	}
	if n > len(prices) || n <= 0 {
		return res, ErrInvalidPos
	}
	return prices[n-1], nil
}

func avgNM(prices []decimal.Decimal, n, m int) (decimal.Decimal, error) {
	avg := decimal.RequireFromString("0")
	if len(prices) <= 0 {
		return avg, ErrEmptyPrices
	}
	if n < 1 || m < 1 || m < n || m > len(prices) {
		return avg, ErrInvalidRange
	}
	if n == m {
		return prices[n-1], nil
	}
	for i := n - 1; i < m; i++ {
		avg = avg.Add(prices[i])
	}
	return avg.DivRound(decimal.NewFromInt(int64(m-n+1)), 8), nil
}

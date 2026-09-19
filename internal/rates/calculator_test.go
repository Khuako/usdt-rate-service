package rates

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestCalculate(t *testing.T) {
	var wantErr decimal.Decimal
	topNPrices := []decimal.Decimal{
		decimal.RequireFromString("0.9991"),
		decimal.RequireFromString("0.9992"),
		decimal.RequireFromString("0.9993"),
	}
	avgNMPrices := []decimal.Decimal{
		decimal.RequireFromString("0.9991"),
		decimal.RequireFromString("0.9992"),
		decimal.RequireFromString("0.9993"),
		decimal.RequireFromString("0.9994"),
		decimal.RequireFromString("0.9995"),
		decimal.RequireFromString("0.9996"),
	}
	roundingTestPrices := []decimal.Decimal{
		decimal.RequireFromString("1"),
		decimal.RequireFromString("1"),
		decimal.RequireFromString("2"),
	}
	var errWant decimal.Decimal
	tests := []struct {
		name    string
		prices  []decimal.Decimal
		calc    Calculation
		want    decimal.Decimal
		wantErr error
	}{
		{"topN first pos", topNPrices, Calculation{Method: MethodTopN, N: 1}, decimal.RequireFromString("0.9991"), nil},
		{"topN middle pos", topNPrices, Calculation{Method: MethodTopN, N: 2}, decimal.RequireFromString("0.9992"), nil},
		{"topN las pos", topNPrices, Calculation{Method: MethodTopN, N: 3}, decimal.RequireFromString("0.9993"), nil},
		{"topN zero pos", topNPrices, Calculation{Method: MethodTopN, N: 0}, errWant, ErrInvalidPos},
		{"topN negative pos", topNPrices, Calculation{Method: MethodTopN, N: -1}, errWant, ErrInvalidPos},
		{"topN beyond pos", topNPrices, Calculation{Method: MethodTopN, N: 7}, errWant, ErrInvalidPos},
		{"topN empty array", []decimal.Decimal{}, Calculation{Method: MethodTopN, N: 1}, errWant, ErrEmptyPrices},
		{name: "empty slice", prices: []decimal.Decimal{}, calc: Calculation{Method: MethodAvgNM, N: 1, M: 10}, want: wantErr, wantErr: ErrEmptyPrices},
		{name: "avgNM all slice", prices: avgNMPrices, calc: Calculation{Method: MethodAvgNM, N: 1, M: 6}, want: decimal.RequireFromString("0.99935"), wantErr: nil},
		{name: "avgNM part slice", prices: avgNMPrices, calc: Calculation{Method: MethodAvgNM, N: 2, M: 4}, want: decimal.RequireFromString("0.9993"), wantErr: nil},
		{name: "avgNM first pos", prices: avgNMPrices, calc: Calculation{Method: MethodAvgNM, N: 1, M: 1}, want: decimal.RequireFromString("0.9991"), wantErr: nil},
		{name: "avgNM last pos", prices: avgNMPrices, calc: Calculation{Method: MethodAvgNM, N: 6, M: 6}, want: decimal.RequireFromString("0.9996"), wantErr: nil},
		{name: "avgNM n is invalid", prices: avgNMPrices, calc: Calculation{Method: MethodAvgNM, N: -1, M: 6}, want: wantErr, wantErr: ErrInvalidRange},
		{name: "avgNM n is zero", prices: avgNMPrices, calc: Calculation{Method: MethodAvgNM, N: 0, M: 6}, want: wantErr, wantErr: ErrInvalidRange},
		{name: "avgNM m is invalid", prices: avgNMPrices, calc: Calculation{Method: MethodAvgNM, N: 1, M: -6}, want: wantErr, wantErr: ErrInvalidRange},
		{name: "avgNM m is less than m", prices: avgNMPrices, calc: Calculation{Method: MethodAvgNM, N: 4, M: 3}, want: wantErr, wantErr: ErrInvalidRange},
		{name: "avgNM m is beyond the slice", prices: avgNMPrices, calc: Calculation{Method: MethodAvgNM, N: 1, M: 10}, want: wantErr, wantErr: ErrInvalidRange},
		{name: "avgNM rounding up", prices: roundingTestPrices, calc: Calculation{Method: MethodAvgNM, N: 1, M: 3}, want: decimal.RequireFromString("1.33333333"), wantErr: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := Calculate(tt.prices, tt.calc)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("wrong error, expected: %v, got: %v", tt.wantErr, err)
			}
			if !val.Equal(tt.want) {
				t.Fatalf("wrong result. expected: %v, got: %v", tt.want, val)
			}
		})
	}
}

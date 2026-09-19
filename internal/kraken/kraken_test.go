package kraken

import (
	"errors"
	"testing"

	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/shopspring/decimal"
)

func TestParseOrderBook(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    rates.OrderBook
		wantErr error
	}{
		{
			name: "valid order book preserves prices and order",
			data: []byte(`{
				"error": [],
				"result": {
					"USDTZUSD": {
						"asks": [["0.9992", "100", 1789673500], ["0.9993", "200", 1789673501]],
						"bids": [["0.9991", "150", 1789673500], ["0.9990", "250", 1789673501]]
					}
				}
			}`),
			want: rates.OrderBook{
				Asks: []decimal.Decimal{
					decimal.RequireFromString("0.9992"),
					decimal.RequireFromString("0.9993"),
				},
				Bids: []decimal.Decimal{
					decimal.RequireFromString("0.9991"),
					decimal.RequireFromString("0.9990"),
				},
			},
		},
		{
			name:    "malformed JSON",
			data:    []byte(`{"error": [], "result":`),
			wantErr: ErrInvalidResponse,
		},
		{
			name:    "exchange error",
			data:    []byte(`{"error": ["EQuery:Unknown asset pair"]}`),
			wantErr: ErrExchange,
		},
		{
			name:    "missing order book",
			data:    []byte(`{"error": [], "result": {}}`),
			wantErr: ErrEmptyResponse,
		},
		{
			name: "empty asks",
			data: []byte(`{"error": [], "result": {"USDTZUSD": {
				"asks": [], "bids": [["0.9991", "150", 1789673500]]
			}}}`),
			wantErr: ErrEmptySide,
		},
		{
			name: "empty bids",
			data: []byte(`{"error": [], "result": {"USDTZUSD": {
				"asks": [["0.9992", "100", 1789673500]], "bids": []
			}}}`),
			wantErr: ErrEmptySide,
		},
		{
			name: "level without a price",
			data: []byte(`{"error": [], "result": {"USDTZUSD": {
				"asks": [[]], "bids": [["0.9991", "150", 1789673500]]
			}}}`),
			wantErr: ErrInvalidPrice,
		},
		{
			name: "price has wrong JSON type",
			data: []byte(`{"error": [], "result": {"USDTZUSD": {
				"asks": [[true, "100", 1789673500]], "bids": [["0.9991", "150", 1789673500]]
			}}}`),
			wantErr: ErrInvalidPrice,
		},
		{
			name: "price is not a decimal",
			data: []byte(`{"error": [], "result": {"USDTZUSD": {
				"asks": [["oops", "100", 1789673500]], "bids": [["0.9991", "150", 1789673500]]
			}}}`),
			wantErr: ErrInvalidPrice,
		},
		{
			name: "zero price",
			data: []byte(`{"error": [], "result": {"USDTZUSD": {
				"asks": [["0", "100", 1789673500]], "bids": [["0.9991", "150", 1789673500]]
			}}}`),
			wantErr: ErrInvalidPrice,
		},
		{
			name: "negative bid price",
			data: []byte(`{"error": [], "result": {"USDTZUSD": {
				"asks": [["0.9992", "100", 1789673500]], "bids": [["-0.5", "150", 1789673500]]
			}}}`),
			wantErr: ErrInvalidPrice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderBook, err := parseOrderBook(tt.data)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("invalid type error. got: %v, want: %v", err, tt.wantErr)
			}
			if len(orderBook.Bids) != len(tt.want.Bids) {
				t.Fatalf(
					"wrong length of bids. got: %d, want: %d",
					len(orderBook.Bids),
					len(tt.want.Bids),
				)
			}
			if len(orderBook.Asks) != len(tt.want.Asks) {
				t.Fatalf(
					"wrong length of ask. got: %d, want: %d",
					len(orderBook.Asks),
					len(tt.want.Asks),
				)
			}
			for i := range len(orderBook.Asks) {
				if !orderBook.Asks[i].Equal(tt.want.Asks[i]) {
					t.Fatalf(
						"wrong ask price, got: %v, want: %v",
						orderBook.Asks[i],
						tt.want.Asks[i],
					)
				}
			}
			for i := range len(orderBook.Bids) {
				if !orderBook.Bids[i].Equal(tt.want.Bids[i]) {
					t.Fatalf(
						"wrong bid price, got: %v, want: %v",
						orderBook.Bids[i],
						tt.want.Bids[i],
					)
				}
			}
		})
	}
}

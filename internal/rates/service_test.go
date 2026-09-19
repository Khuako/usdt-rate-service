package rates_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/Khuako/usdt-rate-service/internal/testutils"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type orderBookProviderStub struct {
	book  rates.OrderBook
	calls int
}

func (p *orderBookProviderStub) GetOrderBook(context.Context) (rates.OrderBook, error) {
	p.calls++
	return p.book, nil
}

type recordingRepository struct {
	attemptedRate rates.Rate
	saveCalls     int
	saveErr       error
}

func (r *recordingRepository) Save(_ context.Context, rate rates.Rate) error {
	r.attemptedRate = rate
	r.saveCalls++
	return r.saveErr
}

func TestService_GetRates_Success(t *testing.T) {
	tests := []struct {
		name    string
		calc    rates.Calculation
		wantAsk string
		wantBid string
	}{
		{
			name:    "topN",
			calc:    rates.Calculation{Method: rates.MethodTopN, N: 2},
			wantAsk: "1.002",
			wantBid: "0.998",
		},
		{
			name:    "avgNM",
			calc:    rates.Calculation{Method: rates.MethodAvgNM, N: 2, M: 3},
			wantAsk: "1.0025",
			wantBid: "0.9975",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &orderBookProviderStub{book: testOrderBook()}
			repo := &recordingRepository{}
			service := rates.NewService(repo, provider)

			got, err := service.GetRates(context.Background(), tt.calc)
			if err != nil {
				t.Fatalf("GetRates: %v", err)
			}
			if got.ID == uuid.Nil {
				t.Fatal("GetRates returned an empty ID")
			}
			want := rates.Rate{
				ID:          got.ID,
				Ask:         decimal.RequireFromString(tt.wantAsk),
				Bid:         decimal.RequireFromString(tt.wantBid),
				ReceivedAt:  provider.book.ReceivedAt,
				Calculation: tt.calc,
			}
			testutils.AssertRateEqual(t, got, want)
			if provider.calls != 1 {
				t.Errorf("provider calls: got %d, want 1", provider.calls)
			}
			if repo.saveCalls != 1 {
				t.Fatalf("Save calls: got %d, want 1", repo.saveCalls)
			}
			testutils.AssertRateEqual(t, repo.attemptedRate, got)
		})
	}
}

func TestService_GetRates_SaveError(t *testing.T) {
	saveErr := errors.New("database unavailable")
	provider := &orderBookProviderStub{book: testOrderBook()}
	repo := &recordingRepository{saveErr: saveErr}
	service := rates.NewService(repo, provider)
	calc := rates.Calculation{Method: rates.MethodTopN, N: 2}

	got, err := service.GetRates(context.Background(), calc)
	if !errors.Is(err, repo.saveErr) {
		t.Errorf("wrong type error, want: %v, got: %v", repo.saveErr, err)
	}

	want := rates.Rate{}

	testutils.AssertRateEqual(t, got, want)
	if repo.saveCalls != 1 {
		t.Fatalf("Save calls: got %d, want 1", repo.saveCalls)
	}
}

func TestService_GetRates_InvalidMethod(t *testing.T) {
	provider := &orderBookProviderStub{book: testOrderBook()}
	repo := &recordingRepository{}
	service := rates.NewService(repo, provider)
	calc := rates.Calculation{Method: rates.Method("Unknown"), N: 2}
	got, err := service.GetRates(context.Background(), calc)
	if !errors.Is(err, rates.ErrWrongCalculationMethod) {
		t.Errorf("wrong type error, want: %v, got: %v", rates.ErrWrongCalculationMethod, err)
	}
	want := rates.Rate{}

	testutils.AssertRateEqual(t, got, want)
	if provider.calls != 0 {
		t.Errorf("too many provider calls: %d", provider.calls)
	}
	if repo.saveCalls != 0 {
		t.Errorf("too many repo calls: %d", repo.saveCalls)
	}
}

func TestService_GetRates_BidCalculationError(t *testing.T) {
	book := testOrderBook()
	book.Bids = book.Bids[:1]
	provider := &orderBookProviderStub{book: book}
	repo := &recordingRepository{}
	service := rates.NewService(repo, provider)

	got, err := service.GetRates(context.Background(), rates.Calculation{Method: rates.MethodTopN, N: 2})
	if !errors.Is(err, rates.ErrInvalidPos) {
		t.Fatalf("GetRates error: got %v, want %v", err, rates.ErrInvalidPos)
	}
	testutils.AssertRateEqual(t, got, rates.Rate{})
	if provider.calls != 1 {
		t.Errorf("provider calls: got %d, want 1", provider.calls)
	}
	if repo.saveCalls != 0 {
		t.Errorf("Save calls: got %d, want 0", repo.saveCalls)
	}
}

func testOrderBook() rates.OrderBook {
	return rates.OrderBook{
		Asks: []decimal.Decimal{
			decimal.RequireFromString("1.001"),
			decimal.RequireFromString("1.002"),
			decimal.RequireFromString("1.003"),
		},
		Bids: []decimal.Decimal{
			decimal.RequireFromString("0.999"),
			decimal.RequireFromString("0.998"),
			decimal.RequireFromString("0.997"),
		},
		ReceivedAt: time.Date(2026, time.September, 19, 12, 0, 0, 0, time.UTC),
	}
}

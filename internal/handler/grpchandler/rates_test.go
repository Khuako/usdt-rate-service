package grpchandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	ratespb "github.com/Khuako/usdt-rate-service/gen/rates"
	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceStub struct {
	result rates.Rate
	err    error
	calc   rates.Calculation
	calls  int
}

func TestHandler_GetRates_ServiceErrors(t *testing.T) {
	internalErr := errors.New("database connection failed: private connection details")
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"invalid position", rates.ErrInvalidPos, codes.InvalidArgument},
		{"invalid range", rates.ErrInvalidRange, codes.InvalidArgument},
		{"canceled", context.Canceled, codes.Canceled},
		{"deadline exceeded", context.DeadlineExceeded, codes.DeadlineExceeded},
		{"internal error", internalErr, codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &serviceStub{err: fmt.Errorf("service: %w", tt.err)}
			handler := NewHandler(service, zap.NewNop())
			got, err := handler.GetRates(t.Context(), &ratespb.GetRatesRequest{
				Method: ratespb.Method_TOP_N,
				N:      2,
			})
			if code := status.Code(err); code != tt.wantCode {
				t.Fatalf("status code: got %v, want %v (error: %v)", code, tt.wantCode, err)
			}
			if got != nil {
				t.Errorf("response: got %v, want nil", got)
			}
			if service.calls != 1 {
				t.Errorf("service calls: got %d, want 1", service.calls)
			}
			if tt.wantCode == codes.Internal && strings.Contains(status.Convert(err).Message(), internalErr.Error()) {
				t.Error("response exposes internal error details")
			}
		})
	}
}

func TestHandler_GetRates_InvalidMethod(t *testing.T) {
	tests := []struct {
		name   string
		method ratespb.Method
	}{
		{"unspecified", ratespb.Method_UNSPECIFIED},
		{"unknown", ratespb.Method(99)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &serviceStub{}
			handler := NewHandler(service, zap.NewNop())
			got, err := handler.GetRates(t.Context(), &ratespb.GetRatesRequest{
				Method: tt.method,
				N:      2,
			})
			if code := status.Code(err); code != codes.InvalidArgument {
				t.Fatalf("status code: got %v, want InvalidArgument (error: %v)", code, err)
			}
			if got != nil {
				t.Errorf("response: got %v, want nil", got)
			}
			if service.calls != 0 {
				t.Errorf("service calls: got %d, want 0", service.calls)
			}
		})
	}
}

func (s *serviceStub) GetRates(_ context.Context, calc rates.Calculation) (rates.Rate, error) {
	s.calc = calc
	s.calls++
	return s.result, s.err
}

func TestHandler_GetRates_Success(t *testing.T) {
	tests := []struct {
		name     string
		request  *ratespb.GetRatesRequest
		wantCalc rates.Calculation
	}{
		{
			name:     "topN",
			request:  &ratespb.GetRatesRequest{Method: ratespb.Method_TOP_N, N: 2},
			wantCalc: rates.Calculation{Method: rates.MethodTopN, N: 2},
		},
		{
			name:     "avgNM",
			request:  &ratespb.GetRatesRequest{Method: ratespb.Method_AVG_NM, N: 2, M: 4},
			wantCalc: rates.Calculation{Method: rates.MethodAvgNM, N: 2, M: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &serviceStub{
				result: rates.Rate{
					ID:          uuid.New(),
					Ask:         decimal.RequireFromString("1.002"),
					Bid:         decimal.RequireFromString("0.998"),
					ReceivedAt:  time.Date(2026, 9, 19, 12, 0, 0, 123456000, time.UTC),
					Calculation: tt.wantCalc,
				},
			}
			handler := NewHandler(service, zap.NewNop())

			got, err := handler.GetRates(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("GetRates: %v", err)
			}
			if service.calls != 1 {
				t.Errorf("service calls: got %d, want 1", service.calls)
			}
			if service.calc != tt.wantCalc {
				t.Errorf("calculation: got %+v, want %+v", service.calc, tt.wantCalc)
			}
			if got == nil || got.Rate == nil {
				t.Fatal("GetRates returned an empty response")
			}
			if got.Rate.Id != service.result.ID.String() {
				t.Errorf("ID: got %q, want %q", got.Rate.Id, service.result.ID.String())
			}
			if got.Rate.Ask != "1.002" {
				t.Errorf("Ask: got %q, want 1.002", got.Rate.Ask)
			}
			if got.Rate.Bid != "0.998" {
				t.Errorf("Bid: got %q, want 0.998", got.Rate.Bid)
			}
			if got.Rate.ReceivedAt == nil {
				t.Fatal("ReceivedAt is nil")
			}
			if !got.Rate.ReceivedAt.AsTime().Equal(service.result.ReceivedAt) {
				t.Errorf("ReceivedAt: got %v, want %v", got.Rate.ReceivedAt.AsTime(), service.result.ReceivedAt)
			}
		})
	}
}

package testutils

import (
	"testing"

	"github.com/Khuako/usdt-rate-service/internal/rates"
)

func AssertRateEqual(t *testing.T, got, want rates.Rate) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("ID: got %v, want %v", got.ID, want.ID)
	}
	if !got.Ask.Equal(want.Ask) {
		t.Errorf("Ask: got %v, want %v", got.Ask, want.Ask)
	}
	if !got.Bid.Equal(want.Bid) {
		t.Errorf("Bid: got %v, want %v", got.Bid, want.Bid)
	}
	if !got.ReceivedAt.Equal(want.ReceivedAt) {
		t.Errorf("ReceivedAt: got %v, want %v", got.ReceivedAt, want.ReceivedAt)
	}
	if got.Calculation != want.Calculation {
		t.Errorf("Calculation: got %+v, want %+v", got.Calculation, want.Calculation)
	}
}

package kraken

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestClient_GetOrderBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.Method != http.MethodGet {
			t.Errorf("wrong method. got: %v, want: get", r.Method)
		}
		if r.URL.Path != "/0/public/Depth" {
			t.Errorf("wrong path: got %q, want %q", r.URL.Path, "/0/public/Depth")
		}
		pairParam := r.URL.Query().Get("pair")
		if pairParam != "USDTUSD" {
			t.Errorf("wrong param. want: USDTUSD, got: %q", pairParam)
		}
		countParam := r.URL.Query().Get("count")
		if countParam != "100" {
			t.Errorf("wrong param. want: 100, got: %q", countParam)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
                "error": [],
                "result": {
                    "USDTZUSD": {
                        "asks": [["1.001", "100", 1789673500]],
                        "bids": [["0.999", "200", 1789673500]]
                    }
                }
            }`))
	}))
	defer server.Close()
	client := NewClient(server.URL)
	before := time.Now().UTC()

	got, err := client.GetOrderBook(context.Background())
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("GetOrderBook: %v", err)
	}
	if got.ReceivedAt.Before(before) || got.ReceivedAt.After(after) {
		t.Fatalf("ReceivedAt %v is outside [%v, %v]", got.ReceivedAt, before, after)
	}
	if len(got.Asks) != 1 {
		t.Fatalf("wrong length of asks, got: %d, want: 1", len(got.Asks))
	}
	if len(got.Bids) != 1 {
		t.Fatalf("wrong length of bids, got: %d, want: 1", len(got.Bids))
	}
	if !got.Asks[0].Equal(decimal.RequireFromString("1.001")) {
		t.Errorf("wrong asks, want: 1.001, want: %v", got.Asks[0])
	}
	if !got.Bids[0].Equal(decimal.RequireFromString("0.999")) {
		t.Errorf("wrong Bids, want: 0.999, want: %v", got.Bids[0])
	}
}
func TestClient_GetOrderBook_HTTP_error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{
                "error": [],
                "result": {
                    "USDTZUSD": {
                        "asks": [["1.001", "100", 1789673500]],
                        "bids": [["0.999", "200", 1789673500]]
                    }
                }
            }`))
	}))
	defer server.Close()
	client := NewClient(server.URL)
	_, err := client.GetOrderBook(context.Background())
	if err == nil {
		t.Fatalf("error is nil")
	}
}
func TestClient_GetOrderBook_ContextErr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
                "error": [],
                "result": {
                    "USDTZUSD": {
                        "asks": [["1.001", "100", 1789673500]],
                        "bids": [["0.999", "200", 1789673500]]
                    }
                }
            }`))
	}))
	defer server.Close()
	client := NewClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.GetOrderBook(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("wrong error type. got: %v, want: context.Canceled", err)
	}
}

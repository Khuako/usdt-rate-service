package grpchandler

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type fakePinger struct {
	calls int
	err   error
}

func (p *fakePinger) Ping(ctx context.Context) error {
	p.calls++
	return p.err
}

func TestHealthHandler_Check(t *testing.T) {
	testError := errors.New("test error")
	tests := []struct {
		name        string
		serviceName string
		wantCalls   int
		pingErr     error
		wantCode    codes.Code
		wantRes     healthpb.HealthCheckResponse_ServingStatus
	}{
		{name: "empty service", wantRes: healthpb.HealthCheckResponse_SERVING, wantCalls: 1},
		{name: "correct service", serviceName: "rates.RatesService", wantRes: healthpb.HealthCheckResponse_SERVING, wantCalls: 1},
		{name: "database unavailable", serviceName: "rates.RatesService", pingErr: testError, wantCode: codes.OK, wantRes: healthpb.HealthCheckResponse_NOT_SERVING, wantCalls: 1},
		{name: "unknown service", serviceName: "unknown", wantCode: codes.NotFound, wantCalls: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := fakePinger{err: tt.pingErr}
			handler := NewHealthHandler(&repo)
			resp, err := handler.Check(t.Context(), &healthpb.HealthCheckRequest{Service: tt.serviceName})
			if repo.calls != tt.wantCalls {
				t.Errorf("expected %d calls, got: %d", tt.wantCalls, repo.calls)
			}
			if code := status.Code(err); code != tt.wantCode {
				t.Fatalf("expected code: %v, got: %v (error: %v)", tt.wantCode, code, err)
			}
			if tt.wantCode != codes.OK {
				if resp != nil {
					t.Errorf("expected nil response, got: %v", resp)
				}
				return
			}
			if resp == nil {
				t.Fatal("expected health response, got nil")
			}
			if resp.Status != tt.wantRes {
				t.Errorf("expected status: %v, got: %v", tt.wantRes, resp.Status)
			}
		})
	}
}
func TestHealthHandler_Shutdown(t *testing.T) {
	repo := fakePinger{}
	handler := NewHealthHandler(&repo)
	handler.Shutdown()
	resp, err := handler.Check(t.Context(), &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("error should be nil, got: %v", err)
	}
	if resp.Status != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Errorf("status should be not serving, got: %v", resp.Status)
	}
	if repo.calls != 0 {
		t.Errorf("repo calls should be 0, got: %v", repo.calls)
	}
}

package grpchandler

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	healthpb.UnimplementedHealthServer
	pinger Pinger
}

func NewHealthHandler(pinger Pinger) *HealthHandler {
	return &HealthHandler{pinger: pinger}
}

func (h *HealthHandler) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	if req.Service != "" && req.Service != "rates.RatesService" {
		return nil, status.Error(codes.NotFound, "service not found")
	}
	timeoutContext, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	err := h.pinger.Ping(timeoutContext)
	if err != nil {
		return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_NOT_SERVING}, nil
	}
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

package grpchandler

import (
	"context"
	"errors"

	ratespb "github.com/Khuako/usdt-rate-service/gen/rates"
	"github.com/Khuako/usdt-rate-service/internal/rates"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	ratespb.UnimplementedRatesServiceServer
	service RateService
	logger  *zap.Logger
}
type RateService interface {
	GetRates(context.Context, rates.Calculation) (rates.Rate, error)
}

func NewHandler(service RateService, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) GetRates(
	ctx context.Context,
	req *ratespb.GetRatesRequest,
) (*ratespb.GetRatesResponse, error) {
	var resp *ratespb.GetRatesResponse
	var m rates.Method
	switch req.Method {
	case ratespb.Method_TOP_N:
		m = rates.MethodTopN
	case ratespb.Method_AVG_NM:
		m = rates.MethodAvgNM
	default:
		return resp, status.Error(codes.InvalidArgument, "invalid method")
	}
	val, err := h.service.GetRates(ctx, rates.Calculation{
		Method: m,
		N:      int(req.N),
		M:      int(req.M),
	})
	if err != nil {
		if errors.Is(err, rates.ErrInvalidRange) || errors.Is(err, rates.ErrInvalidPos) {
			return resp, status.Error(codes.InvalidArgument, "invalid positions")
		}
		if errors.Is(err, context.Canceled) {
			return resp, status.Error(codes.Canceled, "request canceled")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return resp, status.Error(codes.DeadlineExceeded, "deadline exceeded")
		}
		spanCtx := trace.SpanContextFromContext(ctx)
		traceZap := zap.Skip()
		if spanCtx.IsValid() {
			traceZap = zap.String("trace_id", spanCtx.TraceID().String())
		}
		h.logger.Error("failed to get rates", zap.Error(err), traceZap)

		return resp, status.Error(codes.Internal, "internal server error")
	}
	rate := ratespb.Rate{
		Id:         val.ID.String(),
		Ask:        val.Ask.String(),
		Bid:        val.Bid.String(),
		ReceivedAt: timestamppb.New(val.ReceivedAt),
	}
	resp = &ratespb.GetRatesResponse{Rate: &rate}
	return resp, nil
}

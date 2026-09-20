package grpchandler

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		t := time.Now()
		resp, err := handler(ctx, req)
		var lvl zapcore.Level
		if err != nil {
			lvl = zapcore.WarnLevel
		} else {
			lvl = zapcore.InfoLevel
		}
		spanCtx := trace.SpanContextFromContext(ctx)
		traceZap := zap.Skip()
		if spanCtx.IsValid() {
			traceZap = zap.String("trace_id", spanCtx.TraceID().String())
		}
		logger.Log(
			lvl,
			"gRPC request",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(t)),
			zap.String("code", status.Code(err).String()),
			traceZap,
		)
		return resp, err
	}

}

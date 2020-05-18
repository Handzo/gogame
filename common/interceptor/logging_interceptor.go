package interceptor

import (
	"context"
	"time"

	"github.com/Handzo/gogame/common/log"
	"google.golang.org/grpc"
)

func LoggingServerInterceptor(logger log.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		logger = logger.With(log.String("method", info.FullMethod))

		logger.With(log.String("type", "request")).Info(req)

		reply, err := handler(ctx, req)

		logger.With(
			log.String("type", "response"),
			log.Duration(time.Since(start)),
			log.Error(err),
		).Info(reply)

		return reply, err
	}
}

func LoggingClientInterceptor(logger log.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		logger = logger.With(log.String("method", method))

		logger.With(log.String("type", "request")).Info(req)

		err := invoker(ctx, method, req, reply, cc, opts...)

		logger.With(
			log.String("type", "response"),
			log.Duration(time.Since(start)),
			log.Error(err),
		).Info(reply)

		return err
	}
}

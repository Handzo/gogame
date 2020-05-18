package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var InvalidArgument = status.Error(3, `bad request, can not find remote`)

func RequireMetadataKeyServerInterceptor(key string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, InvalidArgument
		}

		data := md.Get(key)

		if len(data) == 0 {
			return nil, InvalidArgument
		}

		ctx = context.WithValue(ctx, key, data[0])

		return handler(ctx, req)
	}
}

func PropagateMetadataClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return InvalidArgument
		}

		ctx = metadata.NewOutgoingContext(ctx, md)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

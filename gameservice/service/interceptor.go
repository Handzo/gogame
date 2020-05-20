package service

import (
	"context"

	"github.com/Handzo/gogame/gameservice/code"
	"github.com/Handzo/gogame/gameservice/repository"
	"google.golang.org/grpc"
)

func AuthServerInterceptor(repo repository.GameRepository) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if info.FullMethod == "/GameService/OpenSession" {
			return handler(ctx, req)
		}

		session, err := repo.GetOpenedSessionForRemote(ctx, ctx.Value("remote").(string))
		if err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, "player_id", session.PlayerId)

		if session == nil {
			return nil, code.SessionNotFound
		}

		return handler(ctx, req)
	}
}

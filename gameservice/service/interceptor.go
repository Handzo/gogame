package service

import (
	"context"

	"github.com/Handzo/gogame/gameservice/code"
	pb "github.com/Handzo/gogame/gameservice/proto"
	"github.com/Handzo/gogame/gameservice/repository"
	"github.com/Handzo/gogame/gameservice/service/pubsub"
	"google.golang.org/grpc"
)

func AuthServerInterceptor(repo repository.GameRepository, pubsub *pubsub.PubSub) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		remote := ctx.Value("remote").(string)

		if info.FullMethod == "/GameService/OpenSession" {
			res, err := handler(ctx, req)
			if err == nil {
				if bindErr := pubsub.Bind(ctx, remote, res.(*pb.OpenSessionResponse).PlayerId); bindErr != nil {
					return nil, code.BindAdressError
				}
			}
			return res, err
		} else if info.FullMethod == "/GameService/CloseSession" {
			go pubsub.Unbind(ctx, remote)
		}

		session, err := repo.GetOpenedSessionForRemote(ctx, remote)
		if err != nil {
			return nil, err
		}

		if session == nil {
			return nil, code.SessionNotFound
		}

		ctx = context.WithValue(ctx, "player_id", session.PlayerId)

		return handler(ctx, req)
	}
}

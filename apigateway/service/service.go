package service

import (
	"context"

	pb "github.com/Handzo/gogame/apigateway/proto"
	authpb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/log"
	gamepb "github.com/Handzo/gogame/gameservice/proto"
)

func NewApiService(authsvc authpb.AuthServiceClient, gamesvc gamepb.GameServiceClient, logger log.Factory) pb.ApiGatewayServiceServer {
	svc := apiService{
		router:  NewRouter(),
		authsvc: authsvc,
		gamesvc: gamesvc,
		logger:  logger,
	}

	svc.router.Register("SignUp", &authpb.SignUpRequest{}, svc.SignUp)
	svc.router.Register("SignIn", &authpb.SignInRequest{}, svc.SignIn)
	svc.router.Register("OpenSession", &gamepb.OpenSessionRequest{}, svc.OpenSession)
	svc.router.Register("CloseSession", &gamepb.CloseSessionRequest{}, svc.CloseSession)

	return svc
}

type apiService struct {
	router  *GRPCRouter
	authsvc authpb.AuthServiceClient
	gamesvc gamepb.GameServiceClient
	logger  log.Factory
}

func (s apiService) Send(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	return s.router.Route(ctx, req)
}

func (s apiService) Connect(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	s.logger.For(ctx).Info("New connection", log.String("remote", ctx.Value("remote").(string)))
	return &pb.Response{}, nil
}

func (s apiService) Disconnect(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	s.logger.For(ctx).Info("Disconnect", log.String("remote", ctx.Value("remote").(string)))

	// close game session if exists
	_, err := s.gamesvc.CloseSession(ctx, &gamepb.CloseSessionRequest{})
	return &pb.Response{}, err
}

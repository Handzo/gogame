package service

import (
	"net"

	pb "github.com/Handzo/gogame/apigateway/proto"
	authpb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/interceptor"
	"github.com/Handzo/gogame/common/log"
	gamepb "github.com/Handzo/gogame/gameservice/proto"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

type Server struct {
	host    string
	service pb.ApiGatewayServiceServer
	tracer  opentracing.Tracer
	logger  log.Factory
}

func NewServer(host string, authsvc authpb.AuthServiceClient, gamesvc gamepb.GameServiceClient, tracer opentracing.Tracer, logger log.Factory) *Server {
	svc := NewApiService(authsvc, gamesvc, logger)

	return &Server{
		host:    host,
		service: svc,
		tracer:  tracer,
		logger:  logger,
	}
}

func (s *Server) Run() error {
	lis, err := net.Listen("tcp", s.host)
	if err != nil {
		s.logger.Bg().Fatalf("failed to dial: %v", err)
	}

	serveropts := []grpc.UnaryServerInterceptor{
		interceptor.RequireMetadataKeyServerInterceptor("remote"),
		otgrpc.OpenTracingServerInterceptor(s.tracer),
		interceptor.SpanLoggingServerInterceptor(s.logger),
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ChainUnaryServer(serveropts...)))

	pb.RegisterApiGatewayServiceServer(grpcServer, s.service)
	s.logger.Bg().Infof("Starting service %s ...", s.host)
	return grpcServer.Serve(lis)
}

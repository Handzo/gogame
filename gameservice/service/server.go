package service

import (
	"net"

	authpb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/interceptor"
	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	pb "github.com/Handzo/gogame/gameservice/proto"
	"github.com/Handzo/gogame/gameservice/repository"
	"github.com/Handzo/gogame/gameservice/repository/postgres"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"
	"google.golang.org/grpc"
)

type Server struct {
	host    string
	service pb.GameServiceServer
	tracer  opentracing.Tracer
	logger  log.Factory
	repo    repository.GameRepository
}

func NewServer(host string, authsvc authpb.AuthServiceClient, tracer opentracing.Tracer, metricsFactory metrics.Factory, logger log.Factory) *Server {
	repo := postgres.New(
		tracing.New("game-db-pg", metricsFactory, logger),
		logger,
	)
	return &Server{
		host:    host,
		service: NewGameService(authsvc, repo, tracer, metricsFactory, logger),
		tracer:  tracer,
		logger:  logger,
		repo:    repo,
	}
}

func (s *Server) Run() error {
	lis, err := net.Listen("tcp", s.host)
	if err != nil {
		s.logger.Bg().Fatalf("failed to dial: %v", err)
	}

	serveropts := []grpc.UnaryServerInterceptor{
		interceptor.RequireMetadataKeyServerInterceptor("remote"),
		interceptor.SpanLoggingServerInterceptor(s.logger),
		interceptor.AuthServerInterceptor(s.repo),
		otgrpc.OpenTracingServerInterceptor(s.tracer),
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ChainUnaryServer(serveropts...)))

	pb.RegisterGameServiceServer(grpcServer, s.service)
	s.logger.Bg().Infof("Starting service %s ...", s.host)
	return grpcServer.Serve(lis)
}

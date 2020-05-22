package service

import (
	"net"

	authpb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/interceptor"
	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	enginepb "github.com/Handzo/gogame/gameengine/proto"
	pb "github.com/Handzo/gogame/gameservice/proto"
	"github.com/Handzo/gogame/gameservice/repository"
	"github.com/Handzo/gogame/gameservice/repository/postgres"
	"github.com/Handzo/gogame/gameservice/service/pubsub"
	"github.com/go-redis/redis"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"
	"google.golang.org/grpc"
)

type Server struct {
	host       string
	service    pb.GameServiceServer
	tracer     opentracing.Tracer
	logger     log.Factory
	repo       repository.GameRepository
	grpcServer *grpc.Server
}

func NewServer(host string, authsvc authpb.AuthServiceClient, enginesvc enginepb.GameEngineClient, tracer opentracing.Tracer, metricsFactory metrics.Factory, logger log.Factory) *Server {
	var rdb *redis.Client
	{
		rdb = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})

		_, err := rdb.Ping().Result()
		if err != nil {
			logger.Bg().Fatal(err)
		}
	}

	pubsub := pubsub.New(
		rdb,
		tracing.New("game-pubsub-redis", metricsFactory, logger),
		logger,
	)

	repo := postgres.New(
		rdb,
		tracing.New("game-db-pg", metricsFactory, logger),
		logger,
	)

	serveropts := []grpc.UnaryServerInterceptor{
		interceptor.RequireMetadataKeyServerInterceptor("remote"),
		AuthServerInterceptor(repo, pubsub),
		otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads()),
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ChainUnaryServer(serveropts...)))

	return &Server{
		host:       host,
		service:    NewGameService(authsvc, enginesvc, repo, pubsub, tracer, metricsFactory, logger),
		tracer:     tracer,
		logger:     logger,
		repo:       repo,
		grpcServer: grpcServer,
	}
}

func (s *Server) Run() error {
	lis, err := net.Listen("tcp", s.host)
	if err != nil {
		s.logger.Bg().Fatalf("failed to dial: %v", err)
	}

	pb.RegisterGameServiceServer(s.grpcServer, s.service)
	s.logger.Bg().Infof("Starting service %s ...", s.host)
	return s.grpcServer.Serve(lis)
}

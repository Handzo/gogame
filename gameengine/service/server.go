package service

import (
	"net"

	pb "github.com/Handzo/gogame/gameengine/proto"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"

	"github.com/opentracing/opentracing-go"

	"github.com/Handzo/gogame/common/interceptor"
	"github.com/Handzo/gogame/common/log"

	"google.golang.org/grpc"
)

type Server struct {
	host    string
	service pb.GameEngineServer
	tracer  opentracing.Tracer
	logger  log.Factory
}

func NewServer(host string, tracer opentracing.Tracer, logger log.Factory) *Server {
	return &Server{
		host:    host,
		service: NewGameEngine(tracer, logger),
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
		otgrpc.OpenTracingServerInterceptor(s.tracer, otgrpc.LogPayloads()),
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ChainUnaryServer(serveropts...)))

	pb.RegisterGameEngineServer(grpcServer, s.service)
	s.logger.Bg().Infof("Starting service %s ...", s.host)
	return grpcServer.Serve(lis)
}

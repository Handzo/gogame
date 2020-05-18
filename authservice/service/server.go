package service

import (
	"net"

	pb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/interceptor"
	"github.com/Handzo/gogame/common/log"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"
	"google.golang.org/grpc"
)

type Server struct {
	host    string
	service pb.AuthServiceServer
	tracer  opentracing.Tracer
	logger  log.Factory
}

func NewServer(host string, tracer opentracing.Tracer, metricsFactory metrics.Factory, logger log.Factory) *Server {
	return &Server{
		host:    host,
		service: NewService(tracer, metricsFactory, logger),
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
		// interceptor.RequireMetadataKeyServerInterceptor("remote"),
		otgrpc.OpenTracingServerInterceptor(s.tracer),
		interceptor.SpanLoggingServerInterceptor(s.logger),
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ChainUnaryServer(serveropts...)))

	pb.RegisterAuthServiceServer(grpcServer, s.service)
	s.logger.Bg().Infof("Starting service %s ...", s.host)
	return grpcServer.Serve(lis)
}

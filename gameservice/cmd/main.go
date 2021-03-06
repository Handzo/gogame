package main

import (
	"flag"
	"fmt"
	"net"

	authpb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/interceptor"
	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	enginepb "github.com/Handzo/gogame/gameengine/proto"
	"github.com/Handzo/gogame/gameservice/service"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/uber/jaeger-lib/metrics"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"google.golang.org/grpc"
)

var (
	port       = flag.Int("port", 7003, "game service port")
	authport   = flag.Int("auth", 7002, "auth service port")
	engineport = flag.Int("engine", 7004, "game engine service port")
)

func main() {
	logger := log.NewFactory(log.NewEntry()).With(log.String("service", "game"))
	metricsFactory := jprom.New().Namespace(metrics.NSOptions{Name: "gogame", Tags: nil})
	tracer := tracing.New("gameservice", metricsFactory, logger)

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(interceptor.ChainUnaryClient(
			interceptor.PropagateMetadataClientInterceptor(),
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads()),
		)),
	}

	// auth service
	var authsvc authpb.AuthServiceClient
	{
		portStr := net.JoinHostPort("localhost", fmt.Sprintf("%d", *authport))
		conn, err := grpc.Dial(portStr, opts...)
		if err != nil {
			logger.Bg().Fatal(err)
		}
		defer conn.Close()
		authsvc = authpb.NewAuthServiceClient(conn)
	}

	// game engine
	var enginesvc enginepb.GameEngineClient
	{
		portStr := net.JoinHostPort("localhost", fmt.Sprintf("%d", *engineport))
		conn, err := grpc.Dial(portStr, opts...)
		if err != nil {
			logger.Bg().Fatal(err)
		}
		defer conn.Close()
		enginesvc = enginepb.NewGameEngineClient(conn)
	}

	host := net.JoinHostPort("localhost", fmt.Sprintf("%d", *port))
	server := service.NewServer(host, authsvc, enginesvc, tracer, metricsFactory, logger)

	logger.Bg().Fatal(server.Run())
}

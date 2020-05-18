package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/Handzo/gogame/apigateway/service"
	authpb "github.com/Handzo/gogame/authservice/proto"
	"github.com/Handzo/gogame/common/interceptor"
	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	gamepb "github.com/Handzo/gogame/gameservice/proto"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/spf13/cobra"
	"github.com/uber/jaeger-lib/metrics"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"google.golang.org/grpc"
)

var (
	port     int
	authport int
	gameport int
)

func init() {
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 7001, "api gateway service port")
	rootCmd.PersistentFlags().IntVarP(&authport, "auth", "a", 7002, "auth service port")
	rootCmd.PersistentFlags().IntVarP(&gameport, "game", "g", 7003, "game service port")
}

var rootCmd = &cobra.Command{
	Use:   "apigateway",
	Short: "apigateway routes requests to corresponding services",
	Long:  "apigateway routes requests to corresponding services",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.NewFactory(log.NewEntry()).With(log.String("service", "apigateway"))
		tracer := tracing.New("apigateway", jprom.New().Namespace(metrics.NSOptions{Name: "gogame", Tags: nil}), logger)

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
			portStr := net.JoinHostPort("localhost", fmt.Sprintf("%d", authport))
			conn, err := grpc.Dial(portStr, opts...)
			if err != nil {
				logger.Bg().Fatal(err)
			}
			defer conn.Close()
			authsvc = authpb.NewAuthServiceClient(conn)
		}

		// gameservice
		var gamesvc gamepb.GameServiceClient
		{
			portStr := net.JoinHostPort("localhost", fmt.Sprintf("%d", gameport))
			conn, err := grpc.Dial(portStr, opts...)
			if err != nil {
				logger.Bg().Fatal(err)
			}
			defer conn.Close()
			gamesvc = gamepb.NewGameServiceClient(conn)
		}

		server := service.NewServer(
			net.JoinHostPort("localhost", fmt.Sprintf("%d", port)),
			authsvc,
			gamesvc,
			tracer,
			logger,
		)

		return server.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

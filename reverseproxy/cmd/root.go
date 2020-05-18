package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"

	apipb "github.com/Handzo/gogame/apigateway/proto"
	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	"github.com/Handzo/gogame/reverseproxy/server"
	"github.com/go-redis/redis"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/spf13/cobra"
	"github.com/uber/jaeger-lib/metrics"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"google.golang.org/grpc"
)

var (
	port    int
	apiport int
)

func init() {
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 7000, "reverseproxy port")
	rootCmd.PersistentFlags().IntVarP(&apiport, "apiport", "a", 7001, "api service port")

	os.Setenv("JAEGER_AGENT_HOST", "192.168.0.10")
	os.Setenv("JAEGER_AGENT_PORT", "6831")
	// - JAEGER_AGENT_HOST=jaeger
	// - JAEGER_AGENT_PORT=6831
}

var rootCmd = &cobra.Command{
	Use:   "reverseproxy",
	Short: "reverseproxy sends pass all request",
	Long:  "reverseproxy sends pass all request",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.NewFactory(log.NewEntry()).With(log.String("service", "reverseproxy"))
		tracer := tracing.New("reverseproxy", jprom.New().Namespace(metrics.NSOptions{Name: "gogame", Tags: nil}), logger)

		var apisvc apipb.ApiGatewayServiceClient
		{
			opts := []grpc.DialOption{
				grpc.WithInsecure(),
				grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
			}

			apiportStr := fmt.Sprintf("%d", apiport)
			conn, err := grpc.Dial(net.JoinHostPort("localhost", apiportStr), opts...)
			if err != nil {
				logger.Bg().Fatal(err)
			}
			defer conn.Close()

			apisvc = apipb.NewApiGatewayServiceClient(conn)
		}

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

		// reverse proxy
		proxy := server.NewProxyServer(apisvc, rdb, logger, tracer)

		s := &http.Server{
			Addr:    net.JoinHostPort("localhost", fmt.Sprintf("%d", port)),
			Handler: proxy,
		}

		logger.Bg().Infof("Starting service on port %d ...", port)
		logger.Bg().Fatal(s.ListenAndServe())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

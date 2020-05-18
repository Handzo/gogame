package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/Handzo/gogame/authservice/service"

	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	"github.com/spf13/cobra"
	"github.com/uber/jaeger-lib/metrics"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
)

var (
	port int
)

func init() {
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 7002, "auth service port")
}

var rootCmd = &cobra.Command{
	Use:   "authservice",
	Short: "authservice responsible for accounting (creating, validating)",
	Long:  "authservice responsible for accounting (creating, validating)",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.NewFactory(log.NewEntry()).With(log.String("service", "auth"))
		metricsFactory := jprom.New().Namespace(metrics.NSOptions{Name: "gogame", Tags: nil})
		tracer := tracing.New("authservice", metricsFactory, logger)

		host := net.JoinHostPort("localhost", fmt.Sprintf("%d", port))
		server := service.NewServer(host, tracer, metricsFactory, logger)
		return server.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/Handzo/gogame/common/log"
	"github.com/Handzo/gogame/common/tracing"
	"github.com/Handzo/gogame/gameengine/service"
	"github.com/uber/jaeger-lib/metrics"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
)

var (
	port = flag.Int("port", 7004, "game engine port")
)

func main() {
	logger := log.NewFactory(log.NewEntry()).With(log.String("service", "auth"))
	metricsFactory := jprom.New().Namespace(metrics.NSOptions{Name: "gogame", Tags: nil})
	tracer := tracing.New("gameengine", metricsFactory, logger)

	host := net.JoinHostPort("localhost", fmt.Sprintf("%d", *port))
	server := service.NewServer(host, tracer, logger)

	logger.Bg().Fatal(server.Run())
}

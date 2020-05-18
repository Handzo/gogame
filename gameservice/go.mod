module github.com/Handzo/gogame/gameservice

go 1.14

replace github.com/Handzo/gogame/common => ../common

replace github.com/Handzo/gogame/authservice => ../authservice

require (
	github.com/Handzo/gogame/authservice v0.0.0-00010101000000-000000000000
	github.com/Handzo/gogame/common v0.0.0-20200514090337-c88bf53402b4
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/go-pg/pg/v9 v9.1.6
	github.com/go-redis/redis v6.15.7+incompatible
	github.com/golang/protobuf v1.4.2
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/kr/pretty v0.2.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/common v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.5.0 // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible
	go.uber.org/atomic v1.5.1 // indirect
	golang.org/x/lint v0.0.0-20200130185559-910be7a94367 // indirect
	golang.org/x/sys v0.0.0-20200217220822-9197077df867 // indirect
	golang.org/x/tools v0.0.0-20200227222343-706bc42d1f0d // indirect
	google.golang.org/genproto v0.0.0-20200218151345-dad8c97a84f5 // indirect
	google.golang.org/grpc v1.29.1
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

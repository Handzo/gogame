module github.com/Handzo/gogame/gameengine

go 1.14

replace github.com/Handzo/gogame/common => ../common

require (
	github.com/Handzo/gogame/common v0.0.0-00010101000000-000000000000
	github.com/golang/protobuf v1.4.2
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/opentracing/opentracing-go v1.1.0
	github.com/prometheus/client_golang v1.6.0 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/uber/jaeger-client-go v2.23.1+incompatible // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	go.uber.org/atomic v1.6.0 // indirect
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55
	google.golang.org/grpc v1.29.1
)

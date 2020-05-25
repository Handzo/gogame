module github.com/Handzo/gogame/apigateway

go 1.14

require (
	github.com/golang/protobuf v1.4.2
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/opentracing/opentracing-go v1.1.0
	github.com/spf13/cobra v1.0.0
	github.com/uber/jaeger-client-go v2.23.1+incompatible // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible
	google.golang.org/grpc v1.29.1
)

replace github.com/Handzo/gogame/common => ../common

replace github.com/Handzo/gogame/authservice => ../authservice

replace github.com/Handzo/gogame/gameservice => ../gameservice

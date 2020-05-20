module github.com/Handzo/gogame/authservice

go 1.14

replace github.com/Handzo/gogame/common => ../common

require (
	github.com/Handzo/gogame/common v0.0.0-00010101000000-000000000000
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-pg/pg/v9 v9.1.6
	github.com/golang/protobuf v1.4.2
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/opentracing/opentracing-go v1.1.0
	github.com/spf13/cobra v1.0.0
	github.com/uber/jaeger-client-go v2.23.1+incompatible // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	google.golang.org/grpc v1.29.1
)

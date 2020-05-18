executable := apigateway

all: test $(executable) run

clean:
	@rm -rf bin/*

protogen:
	protoc proto/api.proto --go_out=plugins=grpc:.

$(executable): protogen
	@echo Building $(executable)
	GO111MODULE=on go build -o bin/$@ -v

test:

run:
	./bin/$(executable)

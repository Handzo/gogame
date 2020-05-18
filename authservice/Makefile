executable := authservice

all: test $(executable) run

clean:
	@rm -rf bin/*

protogen:
	protoc proto/auth.proto --go_out=plugins=grpc:.

$(executable): protogen
	@echo Building $(executable)
	GO111MODULE=on go build -o bin/$@ -v

test:

run:
	./bin/$(executable)

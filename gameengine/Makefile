executable := gameengine

all: test $(executable) run

clean:
	@rm -rf bin/*

protogen:
	protoc proto/engine.proto --go_out=plugins=grpc:.

$(executable): protogen
	@echo Building $(executable)
	GO111MODULE=on go build -o bin/$@ -v cmd/main.go

test:

run:
	./bin/$(executable)

executable := gameservice

all: test $(executable) run

clean:
	@rm -rf bin/*

protogen:
	protoc proto/game.proto --go_out=plugins=grpc:.

$(executable): protogen
	@echo Building $(executable)
	GO111MODULE=on go build -o bin/$@ -v cmd/main.go

test:

run:
	./bin/$(executable)

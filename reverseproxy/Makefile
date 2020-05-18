executable := reverseproxy

all: test $(executable) run

clean:
	@rm -rf bin/*

$(executable):
	@echo Building $(executable)
	GO111MODULE=on go build -o bin/$@ -v

test:

run:
	./bin/$(executable)

.PHONY: all build test

all: build test clean

build:
	go build -o build/twirp-swagger-gen main.go

test:
	./build/twirp-swagger-gen -in example/example.proto -out example/example.swagger.json -host test.example.com
	./build/twirp-swagger-gen -in example/google_timestamp.proto -out example/google_timestamp.swagger.json -host test.example.com

clean:
	go fmt ./...
	go mod tidy
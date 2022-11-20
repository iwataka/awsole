all: fmt lint build

fmt:
	go fmt

lint:
	golangci-lint run

build:
	go build

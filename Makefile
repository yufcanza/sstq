.PHONY: build test bench demo

BINARY_NAME=sttq.exe

build:
	go build -o $(BINARY_NAME) ./cmd/sttq

test: 
	go test -v ./...
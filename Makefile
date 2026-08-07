.PHONY: build test bench benchtime

BINARY_NAME=sttq.exe

build:
	go build -o $(BINARY_NAME) ./cmd/sttq

test: 
	go test -v ./...

bench:
	go test ./... -bench=. -benchmem -benchtime=1x

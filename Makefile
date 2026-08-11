.PHONY: build test bench benchtime

BINARY_NAME=sttq.exe

build:
	go build -o $(BINARY_NAME) ./cmd/sttq

test: 
	go test -v ./...

golden:
	go test ./internal/report/golden_test.go -update

bench:
	go test ./... -bench=. -benchmem -benchtime=1x

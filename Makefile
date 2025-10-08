.PHONY: all build test lint gen

all: build

build:
	go build ./...

test:
	go test ./...

lint:
	@gofmt -l . && test -z "$$(gofmt -l .)"

gen:
	go run ./cmd/erm gen

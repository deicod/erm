.PHONY: all build test lint gen release

all: build

build:
	go build ./...

test:
	go test ./...

lint:
	@gofmt -l . && test -z "$$(gofmt -l .)"

gen:
	go run ./cmd/erm gen

release:
	goreleaser release --snapshot --clean

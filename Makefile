.PHONY: all build test test-race lint gen release

all: build

build:
	go build ./...

test:
        go test ./...

test-race:
        go run ./cmd/erm test --race

lint:
	@gofmt -l . && test -z "$$(gofmt -l .)"

gen:
	go run ./cmd/erm gen

release:
	goreleaser release --snapshot --clean

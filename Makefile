BINARY_NAME=worklog
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: all build test clean run lint

all: test build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) main.go

test:
	go test -v -race ./...

bench:
	go test -bench=. ./...

clean:
	rm -f $(BINARY_NAME)

install:
	go install $(LDFLAGS)

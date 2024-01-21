MODULE = $(shell env GO111MODULE=on go list -m)

VERSION ?= $(shell git describe --tags --always --match='v*' 2> /dev/null || echo v0)
VERSION_HASH = $(shell git rev-parse HEAD)

LDFLAGS += -X "main.Version=$(VERSION)" -X "main.CommitSHA=$(VERSION_HASH)"

## Build:

all: build

build:
	go build -ldflags '$(LDFLAGS)' -o ./dist/ski ./ski

format:
	find . -name '*.go' -exec gofmt -s -w {} +

lint:
	golangci-lint run --out-format=tab --new-from-rev master ./...

tests:
	go test -race -timeout 210s ./...

.PHONY: build format lint tests
MODULE = $(shell env GO111MODULE=on go list -m)

VERSION ?= $(shell git describe --tags --always --match='v*' 2> /dev/null || echo v0)
VERSION_HASH = $(shell git rev-parse HEAD)

LDFLAGS += -X "$(MODULE)/ctl/consts.Version=$(VERSION)" -X "$(MODULE)/ctl/consts.CommitSHA=$(VERSION_HASH)"

## Build:

all: build

build:
	cd ctl && go build -ldflags '$(LDFLAGS)' -o ../dist/cloudcat && cd ..

format:
	find . -name '*.go' -exec gofmt -s -w {} +

lint:
	golangci-lint run --out-format=tab --new-from-rev master ./...

tests:
	find . -name go.mod -execdir go test -race -timeout 60s ./... \;

.PHONY: build format lint tests
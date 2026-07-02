# Dwarpal build tooling.
#
# Targets:
#   all   — vet, test, build (default)
#   build — compile the dwarpal binary with version metadata
#   test  — run all tests
#   vet   — run go vet

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: all build test vet

all: vet test build

build:
	go build -ldflags "$(LDFLAGS)" -o dwarpal ./cmd/dwarpal

test:
	go test ./...

vet:
	go vet ./...

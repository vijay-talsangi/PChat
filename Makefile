.PHONY: build build-all clean install lint test

BINARY  = bin/pchat
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS = -s -w \
	-X github.com/vijay-talsangi/PChat/cmd/pchat.version=$(VERSION) \
	-X github.com/vijay-talsangi/PChat/cmd/pchat.commit=$(COMMIT) \
	-X github.com/vijay-talsangi/PChat/cmd/pchat.date=$(DATE)

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/pchat

run: build
	./$(BINARY)

clean:
	rm -rf bin dist

install:
	CGO_ENABLED=0 go install -ldflags="$(LDFLAGS)" ./cmd/pchat

# Cross-compile for all target platforms
build-all:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/pchat-linux-amd64 ./cmd/pchat && \
	echo "  -> linux/amd64" && \
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/pchat-linux-arm64 ./cmd/pchat && \
	echo "  -> linux/arm64" && \
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/pchat-darwin-amd64 ./cmd/pchat && \
	echo "  -> darwin/amd64" && \
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/pchat-darwin-arm64 ./cmd/pchat && \
	echo "  -> darwin/arm64" && \
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/pchat-windows-amd64.exe ./cmd/pchat && \
	echo "  -> windows/amd64" && \
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/pchat-windows-arm64.exe ./cmd/pchat && \
	echo "  -> windows/arm64"

test:
	go test ./...

lint:
	go vet ./...

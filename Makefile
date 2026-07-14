.PHONY: build build-all clean install lint test

BINARY  = bin/chat
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS = -s -w \
	-X github.com/vijay-talsangi/PChat/cmd/chat.version=$(VERSION) \
	-X github.com/vijay-talsangi/PChat/cmd/chat.commit=$(COMMIT) \
	-X github.com/vijay-talsangi/PChat/cmd/chat.date=$(DATE)

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/chat

run: build
	./$(BINARY)

clean:
	rm -rf bin dist

install:
	CGO_ENABLED=0 go install -ldflags="$(LDFLAGS)" ./cmd/chat

# Cross-compile for all target platforms
build-all:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/chat-linux-amd64 ./cmd/chat && \
	echo "  -> linux/amd64" && \
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/chat-linux-arm64 ./cmd/chat && \
	echo "  -> linux/arm64" && \
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/chat-darwin-amd64 ./cmd/chat && \
	echo "  -> darwin/amd64" && \
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/chat-darwin-arm64 ./cmd/chat && \
	echo "  -> darwin/arm64" && \
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/chat-windows-amd64.exe ./cmd/chat && \
	echo "  -> windows/amd64" && \
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/chat-windows-arm64.exe ./cmd/chat && \
	echo "  -> windows/arm64"

test:
	go test ./...

lint:
	go vet ./...

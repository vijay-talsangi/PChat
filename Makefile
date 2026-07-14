.PHONY: build run clean install

BINARY=bin/chat

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/chat

run: build
	./$(BINARY)

clean:
	rm -rf bin

install: build
	cp $(BINARY) $(GOPATH)/bin/chat
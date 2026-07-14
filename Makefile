.PHONY: build run clean install

BINARY=chat

build:
	go build -o $(BINARY) ./cmd/chat

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

install: build
	cp $(BINARY) $(GOPATH)/bin/$(BINARY)

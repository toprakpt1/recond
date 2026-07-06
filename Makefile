.PHONY: all build build-daemon build-cli clean run-daemon run-cli

BINARY_DIR = bin

all: build

build: build-daemon build-cli

build-daemon:
	go build -o $(BINARY_DIR)/recond ./cmd/recond

build-cli:
	go build -o $(BINARY_DIR)/recon ./cmd/recon

run-daemon: build-daemon
	./$(BINARY_DIR)/recond

run-cli: build-cli
	./$(BINARY_DIR)/recon

clean:
	rm -rf $(BINARY_DIR)

test:
	go test ./...

lint:
	go vet ./...

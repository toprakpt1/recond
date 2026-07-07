.PHONY: all build build-daemon build-cli clean run-daemon run-cli test lint install uninstall

BINARY_DIR = bin
INSTALL_DIR = /usr/local/bin
VERSION    ?= dev
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE       ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS     = -s -w \
              -X github.com/toprakpt1/recond/internal/version.Version=$(VERSION) \
              -X github.com/toprakpt1/recond/internal/version.Commit=$(COMMIT) \
              -X github.com/toprakpt1/recond/internal/version.Date=$(DATE)

all: build

build: build-daemon build-cli

build-daemon:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_DIR)/recond ./cmd/recond

build-cli:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_DIR)/recon ./cmd/recon

run-daemon: build-daemon
	./$(BINARY_DIR)/recond

run-cli: build-cli
	./$(BINARY_DIR)/recon

install: build
	install -d $(INSTALL_DIR)
	install -m 755 $(BINARY_DIR)/recond $(INSTALL_DIR)/recond
	install -m 755 $(BINARY_DIR)/recon $(INSTALL_DIR)/recon
	@echo "Installed to $(INSTALL_DIR)/recond and $(INSTALL_DIR)/recon"

uninstall:
	rm -f $(INSTALL_DIR)/recond $(INSTALL_DIR)/recon
	@echo "Removed from $(INSTALL_DIR)"

clean:
	rm -rf $(BINARY_DIR)

test:
	go test ./...

lint:
	go vet ./...

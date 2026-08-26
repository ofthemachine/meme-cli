# Makefile for meme-cli

.PHONY: all build test test-integration fmt lint clean install gallery help

BINARY_NAME=meme-cli
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "v0.0.0-dev")
COMMIT_SHA ?= $(shell git rev-parse --verify -q HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -s -w -X github.com/ofthemachine/meme-cli/cmd.version=$(VERSION)+$(COMMIT_SHA)
BUILD_FLAGS = -trimpath -buildvcs=false -ldflags="$(LDFLAGS)"

all: build

help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: fmt lint ## Build the binary for the current platform
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BINARY_NAME) .
	@echo "Binary: $(BINARY_NAME)"

test: ## Run unit tests
	go test ./...

test-integration: ## Run CLI integration tests (builds and drives the real binary)
	go test ./tests -v -tags=integration

fmt: ## Format code
	go fmt ./...

lint: ## Lint code (fails on violations; install: https://golangci-lint.run/welcome/install/)
	golangci-lint run

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	go clean

gallery: ## Regenerate examples/gallery.html review page
	python3 scripts/generate_gallery.py

install: build ## Install binary to GOPATH/bin
	cp $(BINARY_NAME) $(shell go env GOPATH)/bin/

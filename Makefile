# tlgram Makefile

# Variables
BINARY_NAME := tlgram
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

# TDLib paths (adjust as needed)
TDLIB_DIR ?= /opt/tdlib
CGO_CFLAGS := -I$(TDLIB_DIR)/include
CGO_LDFLAGS := -L$(TDLIB_DIR)/lib -ltdjson_static -ltdjson_private -ltdclient -ltdcore -ltdapi -ltdactor -ltddb -ltdsqlite -ltdnet -ltdutils -lstdc++ -lssl -lcrypto -ldl -lz -lm -lpthread

# Go environment
export CGO_ENABLED=1
export CGO_CFLAGS
export CGO_LDFLAGS

.PHONY: all build build-static clean test test-race lint fmt vet check run install tdlib docker-tdlib help

# Default target
all: build

## Build targets

# Build for development
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/tlgram

# Build without TDLib (for testing UI components)
build-no-tdlib:
	@echo "Building $(BINARY_NAME) without TDLib..."
	@mkdir -p bin
	CGO_ENABLED=0 go build $(LDFLAGS) -tags no_tdlib -o bin/$(BINARY_NAME) ./cmd/tlgram

# Build fully static binary (for distribution)
build-static:
	@echo "Building static $(BINARY_NAME)..."
	@mkdir -p bin
	go build $(LDFLAGS) -ldflags "-linkmode external -extldflags '-static'" -o bin/$(BINARY_NAME)-static ./cmd/tlgram

# Cross-compile for different architectures
build-linux-amd64:
	@echo "Building for linux/amd64..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/tlgram

build-linux-arm64:
	@echo "Building for linux/arm64..."
	@mkdir -p bin
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/tlgram

## Test targets

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race -v ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Lint and format targets

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run all checks
check: fmt vet lint test

## Run targets

# Run the application
run: build
	./bin/$(BINARY_NAME)

# Run with specific chat
run-chat: build
	./bin/$(BINARY_NAME) --chat $(CHAT)

## Install target

# Install to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/

## TDLib targets

# Build TDLib locally
tdlib:
	@echo "Building TDLib..."
	./scripts/build-tdlib.sh

# Build TDLib in Docker (reproducible)
docker-tdlib:
	@echo "Building TDLib in Docker..."
	docker build -f Dockerfile.tdlib -t tlgram-tdlib-builder .
	docker run --rm -v $(PWD)/tdlib-out:/out tlgram-tdlib-builder
	@echo "TDLib built to tdlib-out/"

## Utility targets

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf dist/
	rm -rf tdlib-out/
	rm -f coverage.out coverage.html

# Generate mocks for testing
mocks:
	@echo "Generating mocks..."
	@which mockgen > /dev/null || go install github.com/golang/mock/mockgen@latest
	go generate ./...

# Update dependencies
deps:
	@echo "Updating dependencies..."
	go mod tidy
	go mod download

# Show help
help:
	@echo "tlgram Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build          Build the application"
	@echo "  build-no-tdlib Build without TDLib (for UI testing)"
	@echo "  build-static   Build fully static binary"
	@echo "  build-linux-*  Cross-compile for Linux"
	@echo ""
	@echo "Test targets:"
	@echo "  test           Run tests"
	@echo "  test-race      Run tests with race detector"
	@echo "  test-cover     Run tests with coverage report"
	@echo ""
	@echo "Lint targets:"
	@echo "  lint           Run golangci-lint"
	@echo "  fmt            Format code"
	@echo "  vet            Run go vet"
	@echo "  check          Run all checks (fmt, vet, lint, test)"
	@echo ""
	@echo "Run targets:"
	@echo "  run            Build and run the application"
	@echo "  run-chat       Run with CHAT=@username"
	@echo ""
	@echo "TDLib targets:"
	@echo "  tdlib          Build TDLib locally"
	@echo "  docker-tdlib   Build TDLib in Docker"
	@echo ""
	@echo "Utility targets:"
	@echo "  install        Install to GOPATH/bin"
	@echo "  clean          Clean build artifacts"
	@echo "  deps           Update dependencies"
	@echo "  help           Show this help"

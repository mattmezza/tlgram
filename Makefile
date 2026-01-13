# tlgram Makefile

# Variables
BINARY_NAME := tlgram
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

# Go environment - pure Go, no CGO required
export CGO_ENABLED=0

.PHONY: all build build-static clean test test-race lint fmt vet check run install help

# Default target
all: build

## Build targets

# Build for development
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/tlgram

# Cross-compile for different architectures
build-linux-amd64:
	@echo "Building for linux/amd64..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/tlgram

build-linux-arm64:
	@echo "Building for linux/arm64..."
	@mkdir -p bin
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/tlgram

build-darwin-amd64:
	@echo "Building for darwin/amd64..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/tlgram

build-darwin-arm64:
	@echo "Building for darwin/arm64..."
	@mkdir -p bin
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/tlgram

build-windows-amd64:
	@echo "Building for windows/amd64..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/tlgram

# Build all platforms
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

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
	@echo "  build              Build the application"
	@echo "  build-linux-amd64  Cross-compile for Linux amd64"
	@echo "  build-linux-arm64  Cross-compile for Linux arm64"
	@echo "  build-darwin-amd64 Cross-compile for macOS amd64"
	@echo "  build-darwin-arm64 Cross-compile for macOS arm64 (Apple Silicon)"
	@echo "  build-windows-amd64 Cross-compile for Windows amd64"
	@echo "  build-all          Build for all platforms"
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
	@echo "Utility targets:"
	@echo "  install        Install to GOPATH/bin"
	@echo "  clean          Clean build artifacts"
	@echo "  deps           Update dependencies"
	@echo "  help           Show this help"

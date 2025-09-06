.PHONY: build test clean lint fmt vet install dev check mod help update-extensions

# Variables
BINARY_NAME=pgbox
BUILD_DIR=./dist
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  test       - Run tests"
	@echo "  clean      - Remove build artifacts"
	@echo "  lint       - Run golangci-lint"
	@echo "  fmt        - Format code with go fmt"
	@echo "  vet        - Run go vet"
	@echo "  install    - Install binary to GOPATH/bin"
	@echo "  dev        - Build and run for development"
	@echo "  check      - Run fmt, vet, lint, and test"
	@echo "  mod        - Tidy and verify go modules"
	@echo "  release    - Build release binaries for multiple platforms"
	@echo "  update-extensions - Update extension catalogs from builtin and apt sources"
	@echo "  help       - Show this help message"

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/pgbox

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Install binary
install:
	go install $(LDFLAGS) ./cmd/pgbox

# Development build and run
dev: build
	./$(BINARY_NAME)

# Run all quality checks
check: fmt vet lint test

# Go module management
mod:
	go mod tidy
	go mod verify

# Build release binaries for multiple platforms
release: clean
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/pgbox
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/pgbox
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/pgbox
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/pgbox
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/pgbox

# Update extension catalogs
update-extensions:
	@echo "Updating builtin extensions catalog..."
	./scripts/build-official-extensions-list.bash
	@echo "Updating apt package extensions catalog..."
	./scripts/build-apt-clist.bash

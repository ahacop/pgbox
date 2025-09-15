# pgbox - PostgreSQL-in-Docker with selectable extensions

# Variables
BINARY_NAME := pgbox
GO := go

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.0-dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

# Run tests
.PHONY: test
test:
	$(GO) test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Run go vet
.PHONY: vet
vet:
	$(GO) vet ./...

# Run linter (requires golangci-lint)
.PHONY: lint
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

# Run all checks
.PHONY: check
check: fmt vet test

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Install to GOPATH/bin
.PHONY: install
install:
	$(GO) install -ldflags "$(LDFLAGS)" .

# Development build and run
.PHONY: dev
dev: build
	./$(BINARY_NAME)

# Show version
.PHONY: version
version: build
	./$(BINARY_NAME) --version

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run golangci-lint (must be installed)"
	@echo "  check         - Run fmt, vet, and test"
	@echo "  clean         - Remove build artifacts"
	@echo "  install       - Install to GOPATH/bin"
	@echo "  dev           - Build and run for development"
	@echo "  version       - Show version information"
	@echo "  update-extensions - Update extension catalogs"
	@echo "  help          - Show this help message"

# Update extension catalogs
.PHONY: update-extensions
update-extensions:
	@echo "Updating builtin extensions catalog..."
	./scripts/build-official-extensions-list.bash
	@echo "Updating apt package extensions catalog..."
	./scripts/build-apt-clist.bash
	@echo "Generating extension name mappings..."
	./scripts/build-extension-mappings.bash
	@echo "Merging extension data into single file..."
	./scripts/build-merged-extensions.bash

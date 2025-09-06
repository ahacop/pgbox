# pgbox - PostgreSQL-in-Docker with selectable extensions

# Variables
BINARY_NAME := pgbox
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/ahacop/pgbox/cmd.Version=$(VERSION)"
GO := go

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) .

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
	$(GO) install $(LDFLAGS) .

# Development build and run
.PHONY: dev
dev: build
	./$(BINARY_NAME)

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
	@echo "  update-extensions - Update extension catalogs"
	@echo "  help          - Show this help message"

# Update extension catalogs
.PHONY: update-extensions
update-extensions:
	@echo "Updating builtin extensions catalog..."
	./scripts/build-official-extensions-list.bash
	@echo "Updating apt package extensions catalog..."
	./scripts/build-apt-clist.bash

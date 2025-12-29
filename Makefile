# pgbox - PostgreSQL-in-Docker with selectable extensions

# Variables
BINARY_NAME := pgbox
GO := go

# Extension and PostgreSQL configuration
EXTS ?= pgvector,pg_cron
PG_VERSION ?= 17
PORT ?= 5432

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
	@echo "  build             - Build the binary"
	@echo "  test              - Run tests"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  fmt               - Format code"
	@echo "  vet               - Run go vet"
	@echo "  lint              - Run golangci-lint (must be installed)"
	@echo "  check             - Run fmt, vet, and test"
	@echo "  clean             - Remove build artifacts"
	@echo "  install           - Install to GOPATH/bin"
	@echo "  dev               - Build and run for development"
	@echo "  version           - Show version information"
	@echo "  release           - Create a release with goreleaser"
	@echo "  release-snapshot  - Test release build without publishing"
	@echo "  update-nix-hash   - Update Nix vendorHash after Go module changes"
	@echo "  help              - Show this help message"

# Export Docker configuration with extensions
.PHONY: export
export:
	@mkdir -p out
	$(GO) run . export out --ext $(EXTS) --version $(PG_VERSION)

# Run PostgreSQL with extensions
.PHONY: run
run:
	$(GO) run . up --ext $(EXTS) --version $(PG_VERSION) --port $(PORT)

# Update Nix vendorHash
.PHONY: update-nix-hash
update-nix-hash:
	@./scripts/update-nix-hash.sh

# Create a release with goreleaser
.PHONY: release
release:
	@which goreleaser > /dev/null || (echo "goreleaser not found. Install from https://goreleaser.com/install/" && exit 1)
	goreleaser release --clean

# Test release build without publishing
.PHONY: release-snapshot
release-snapshot:
	@which goreleaser > /dev/null || (echo "goreleaser not found. Install from https://goreleaser.com/install/" && exit 1)
	goreleaser release --snapshot --clean

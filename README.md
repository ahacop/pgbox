# pgbox

PostgreSQL-in-Docker with selectable extensions.

## Overview

pgbox is a CLI tool that simplifies running PostgreSQL in Docker with your choice of extensions. It provides an easy way to spin up PostgreSQL instances with specific extensions for development and testing purposes.

## Installation

### Using Nix Flake (no installation required)

```bash
# Run directly from GitHub
nix run github:ahacop/pgbox -- --help

# Start with specific extensions
nix run github:ahacop/pgbox -- up --ext pgvector,postgis
```

### Build from source

```bash
# Build from source
make build

# Install to GOPATH/bin
make install
```

## Usage

```bash
# Show help
./pgbox --help

# TODO: Add commands here as they are implemented
```

## Development

### Prerequisites

- Go 1.21+
- Docker with compose plugin (for future functionality)
- Optional: golangci-lint for linting

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Run all checks (format, vet, test)
make check

# Clean build artifacts
make clean
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

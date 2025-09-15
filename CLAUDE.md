# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

pgbox is a Go CLI application that simplifies running PostgreSQL in Docker with selectable extensions. It manages Docker containers with PostgreSQL instances and can install extensions from the apt.postgresql.org repository.

## Build and Development Commands

```bash
# Build the binary
make build

# Run tests
make test

# Run a single test
go test -v -run TestSpecificFunction ./internal/docker

# Run tests with coverage
make test-coverage

# Run all quality checks (format, vet, test)
make check

# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Development build and run
make dev

# Clean build artifacts
make clean

# Install binary to GOPATH/bin
make install

# Update extension catalogs from Docker images
make update-extensions

# Generate TOML files from extension data
make generate-toml

# Update Nix vendorHash after Go module changes
make update-nix-hash
```

## Key Commands

```bash
# Start PostgreSQL with extensions
./pgbox up --ext pgvector,hypopg

# Connect to PostgreSQL with psql
./pgbox psql

# Stop containers
./pgbox down

# Clean up all pgbox containers, volumes, and images
./pgbox clean
# Or with auto-confirm: echo "y" | ./pgbox clean

# Export Docker artifacts
./pgbox export ./my-postgres --ext pgvector,pg_cron

# List available extensions
./pgbox list-extensions

# Check container status
./pgbox status

# View container logs
./pgbox logs
```

## Architecture

### Two Extension Configuration Systems

The codebase has TWO extension configuration systems:

1. **Legacy System** (used by `up` command):
   - Uses `extensions.jsonl` embedded file for extension metadata
   - Simple package mapping via `extensions.Manager`
   - Located in `internal/extensions/extensions.go` and `loader.go`
   - Only handles package installation, not configuration

2. **TOML-Based System** (used by `export` and `export-new` commands):
   - Uses TOML files in `extensions/` directory for rich configuration
   - Handles shared_preload_libraries, GUCs, and complex SQL initialization
   - Components:
     - `internal/extspec/`: TOML schema definitions
     - `internal/extensions/toml_loader.go`: TOML file loading
     - `internal/applier/`: Applies TOML specs to Docker/PostgreSQL configs
     - `internal/model/`: Data models for Dockerfile, Compose, PostgreSQL configs
     - `internal/render/`: Renders models to actual files

### Command Layer (`cmd/`)

- **up.go**: Uses legacy system - builds custom Docker images with extensions
- **export.go**: Legacy export using simple template system
- **export_new.go**: Uses TOML system with full configuration support
- **psql.go**: Connects to running PostgreSQL instance
- **down.go**, **status.go**, **logs.go**: Container management

### Extension TOML Structure

Extensions requiring special configuration (e.g., pg_cron) have TOML files:

```toml
extension = "pg_cron"
package = "postgresql-17-cron"

[postgresql.conf]
shared_preload_libraries = ["pg_cron"]
"cron.database_name" = "postgres"

[[sql.initdb]]
text = "CREATE EXTENSION IF NOT EXISTS pg_cron;"
```

### Docker Integration

When extensions are requested:

1. Creates temporary build directory with generated Dockerfile
2. Builds custom image based on postgres:XX with apt packages
3. For extensions with shared_preload_libraries (TOML system only):
   - Configures PostgreSQL startup parameters
   - Sets required GUCs
4. Mounts init.sql for extension creation
5. Uses Docker volumes for data persistence

## Testing Specific Components

```bash
# Test Docker client
go test -v ./internal/docker

# Test extension management
go test -v ./internal/extensions

# Test TOML loading and application
go test -v ./internal/extspec ./internal/applier

# Test with specific extension
./pgbox up --ext pg_cron,pgvector
./pgbox psql -- -c "SELECT * FROM pg_extension;"
```

## Important Notes

- Extensions like `pg_cron`, `wal2json`, `timescaledb` require `shared_preload_libraries`
- The `up` command currently doesn't fully support the TOML configuration system
- When modifying extension support, update both `extensions.jsonl` and TOML files
- Container names follow pattern: `pgbox-pg{version}` by default
- Extension name mapping: some extensions have different SQL names (e.g., "pgvector" → "vector")

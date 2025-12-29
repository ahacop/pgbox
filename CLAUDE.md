# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

pgbox is a Go CLI application that simplifies running PostgreSQL in Docker with selectable extensions. It manages Docker containers with PostgreSQL instances and can install extensions from the apt.postgresql.org repository.

## Project Structure

- **cmd/**: Command implementations (up, down, psql, export, status, logs, restart, clean, list-extensions)
- **internal/**: Core business logic
  - **config/**: PostgreSQL configuration management
  - **container/**: Container lifecycle management and naming
  - **docker/**: Docker command wrapper with interface for testability
  - **extensions/**: Extension catalog (Go map with 150+ extensions)
  - **model/**: Data models for Dockerfile, Compose, PostgreSQL configs
  - **orchestrator/**: Business logic extracted from commands (testable)
  - **render/**: Renders models to Docker artifacts
- **scripts/**: Build scripts

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

### Extension Catalog System

Extensions are defined in a Go map in `internal/extensions/catalog.go`:

```go
var Catalog = map[string]Extension{
    // Built-in extensions (no apt package needed)
    "hstore": {},
    "ltree":  {},

    // Third-party extensions
    "pgvector": {Package: "postgresql-{v}-pgvector", SQLName: "vector"},
    "hypopg":   {Package: "postgresql-{v}-hypopg"},

    // Complex extensions (need shared_preload_libraries and/or GUCs)
    "pg_cron": {
        Package: "postgresql-{v}-cron",
        Preload: []string{"pg_cron"},
        GUCs: map[string]string{
            "cron.database_name": "postgres",
        },
        InitSQL: "CREATE EXTENSION IF NOT EXISTS pg_cron;\nGRANT USAGE ON SCHEMA cron TO postgres;",
    },
}
```

Key functions:
- `extensions.Get(name)` - lookup extension
- `extensions.GetPackage(name, version)` - get apt package name
- `extensions.GetInitSQL(name)` - get initialization SQL
- `extensions.ValidateExtensions(names)` - validate extensions exist
- `extensions.ListExtensions()` - list all extensions

### Docker Integration

The Docker interface (`internal/docker/docker.go`) enables testability:

```go
type Docker interface {
    RunCommand(args ...string) error
    RunCommandWithOutput(args ...string) (string, error)
    RunPostgres(pgConfig *config.PostgresConfig, opts ContainerOptions) error
    // ... etc
}
```

When extensions are requested:
1. Validates extensions exist in catalog
2. Collects apt packages, shared_preload_libraries, GUCs, init SQL
3. Builds custom Docker image if packages needed
4. Mounts init.sql for extension creation
5. Uses Docker volumes for data persistence
6. Container naming: `pgbox-pg{version}-{hash}` based on extensions
7. Image naming: deterministic based on extensions + their configs

### Orchestrator Pattern

Business logic is extracted into `internal/orchestrator/` for testability:

```go
type UpOrchestrator struct {
    docker       docker.Docker
    containerMgr *container.Manager
}

func (o *UpOrchestrator) Run(cfg UpConfig) error {
    // All business logic for starting PostgreSQL
}
```

Commands in `cmd/` are thin wrappers that parse flags and call orchestrators.

## Testing

```bash
# Test Docker client
go test -v ./internal/docker

# Test extension catalog
go test -v ./internal/extensions

# Test orchestrator (with mock Docker)
go test -v ./internal/orchestrator

# Test export command
go test -v ./cmd

# Manual test with specific extension
./pgbox up --ext pg_cron,pgvector
./pgbox psql -- -c "SELECT * FROM pg_extension;"
```

## Important Notes

- Extensions like `pg_cron`, `wal2json` require `shared_preload_libraries`
- To add a new extension, add it to `internal/extensions/catalog.go`
- Container names follow pattern: `pgbox-pg{version}-{hash}` when extensions used
- Extension name mapping: some extensions have different SQL names (e.g., "pgvector" → "vector")
- Default PostgreSQL versions: 16 and 17 (17 is default)
- Default credentials: user=postgres, password=postgres, database=postgres
- Default port: 5432
- Image hash includes extension config, so changes trigger rebuilds

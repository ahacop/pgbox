# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

pgbox is a Go CLI application that provides a wrapper around Docker Compose to run PostgreSQL-in-Docker with selectable extensions. It creates temporary or exportable scaffolding with Dockerfile and docker-compose.yml configurations.

## Build and Development Commands

```bash
# Build the binary
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run all quality checks (format, vet, lint, test)
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

# Build release binaries for multiple platforms
make release
```

## Architecture

The application follows a modular Go architecture with clear separation of concerns:

### Core Components
- **cmd/pgbox/main.go**: CLI entry point using cobra for command handling
- **internal/config/**: Configuration management with XDG directory standards
- **pkg/extensions/**: Extension metadata management from JSON files
- **pkg/scaffold/**: Dynamic generation of Docker artifacts
- **pkg/docker/**: Docker compose operations wrapper
- **pkg/ui/**: Interactive TUI for extension selection using Bubble Tea

### Key Architectural Patterns
- **Configuration-driven**: Uses `internal/config/config.go:21` for centralized configuration
- **Plugin-style extensions**: Extensions loaded from JSON metadata in `pgbox-data/`
- **Temporary scaffolding**: Runtime Docker artifacts created in XDG temp directories
- **State persistence**: Instance tracking in `${XDG_STATE_HOME}/pgbox/`

### Extension System
Extensions are managed through JSON files:
- `pgbox-data/builtin/pg17.json`: Built-in PostgreSQL extensions
- `pgbox-data/apt-bookworm-pgdg/pg17.json`: Apt package extensions

The extension manager (`pkg/extensions/extensions.go:26`) handles:
- Loading extension metadata from JSON files
- Distinguishing between builtin and packaged extensions
- Providing interactive selection via TUI

### Scaffolding System
The scaffold generator (`pkg/scaffold/scaffold.go:14`) creates:
- **Dockerfile**: Dynamically includes required apt packages for extensions
- **docker-compose.yml**: Service definition with proper volume mounts
- **initdb/01-extensions.sql**: SQL to create selected extensions

## Key Commands

### Development and Testing
```bash
# Build and test the application
./pgbox up --ext hypopg,pgvector
./pgbox status
./pgbox psql
./pgbox down --wipe
```

### Common Usage Patterns
```bash
# Start with interactive extension selection
./pgbox up

# Start with specific extensions
./pgbox up --ext hypopg,pgvector,postgis --port 5433 --name mydb

# Export scaffolding for sharing/version control
./pgbox export ./my-postgres-setup

# Connect to running instance
./pgbox psql --db appdb --user appuser
```

## Technical Details

### Dependencies
- Go 1.21+
- Docker with compose plugin
- Optional: `psql` client for database connections

### Key Libraries
- `github.com/spf13/cobra`: CLI framework
- `github.com/charmbracelet/bubbletea`: TUI framework for extension selection
- `github.com/charmbracelet/bubbles`: TUI components

### Default Configuration
- PostgreSQL version: 17 (overrideable via `PG_MAJOR` env var)
- Default database: `appdb`
- Default user: `appuser`
- Default password: `changeme`
- Default port: `5432`
- Instance name: `pgbox`

### Extension Ecosystem
Supports PostgreSQL extensions from two sources:
- **Built-in**: Extensions included with PostgreSQL
- **Packaged**: Extensions from apt.postgresql.org

Common extensions include hypopg, pgvector, postgis, pg_stat_statements, hstore, citext.
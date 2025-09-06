# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

pgbox is a Go CLI application (currently in early development) that will provide a wrapper around Docker Compose to run PostgreSQL-in-Docker with selectable extensions. The goal is to create temporary or exportable scaffolding with Dockerfile and docker-compose.yml configurations.

## Build and Development Commands

```bash
# Build the binary
make build

# Run tests
make test

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
```

## Current Project Structure

The project is in initial development with the following structure:

- **main.go**: Entry point that calls cmd.Execute()
- **cmd/root.go**: Cobra CLI skeleton with Version variable set at build time
- **pgbox-data/**: Extension metadata JSON files
  - `builtin/pg{16,17}.json`: Built-in PostgreSQL extensions
  - `apt-bookworm-pgdg/pg{16,17}.json`: Extensions from apt.postgresql.org
- **scripts/**: Maintenance scripts for updating extension catalogs
  - `build-official-extensions-list.bash`: Updates builtin extension list
  - `build-apt-clist.bash`: Updates apt package extension list

## Planned Architecture

The following components are planned but not yet implemented:

- **internal/config/**: Configuration management with XDG directory standards
- **pkg/extensions/**: Extension metadata management from JSON files
- **pkg/scaffold/**: Dynamic generation of Docker artifacts
- **pkg/docker/**: Docker compose operations wrapper
- **pkg/ui/**: Interactive TUI for extension selection

## Extension Data Format

Extension JSON files contain:
```json
{
  "generated_at": "timestamp",
  "source": "postgres:17-bookworm pg_available_extensions",
  "pg_major": 17,
  "entries": [
    {"name": "extension_name", "kind": "builtin|package", "description": "..."}
  ]
}
```

## Development Notes

- The project uses Cobra for CLI command structure
- Version is injected at build time via ldflags
- Extension catalogs are generated from official PostgreSQL Docker images
- Tests use testify for assertions
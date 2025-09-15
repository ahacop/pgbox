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

### Quick Start

```bash
# Start PostgreSQL with default settings (no extensions)
./pgbox up

# Start PostgreSQL with specific extensions
./pgbox up --ext pgvector,hypopg

# Connect to running PostgreSQL instance
./pgbox psql

# Stop and remove containers
./pgbox down
```

### Common Commands

#### Starting PostgreSQL

```bash
# Start with default settings (PostgreSQL 17, port 5432)
./pgbox up

# Start with specific extensions
./pgbox up --ext pgvector,postgis,pg_trgm

# Start with custom PostgreSQL version
./pgbox up --pg-version 16

# Start on a different port
./pgbox up --port 5433

# Start with custom credentials
./pgbox up --user myuser --password mypass --database mydb

# Start without detaching (see logs in foreground)
./pgbox up --detach=false

# Start with custom container name
./pgbox up --name my-postgres-dev
```

#### Managing Containers

```bash
# Check status of running containers
./pgbox status

# View container logs
./pgbox logs

# Follow logs in real-time
./pgbox logs --follow

# Restart container
./pgbox restart

# Stop container (keeps data)
./pgbox down

# Stop and remove container with volumes
./pgbox down --volumes

# Clean up all pgbox containers and volumes
./pgbox clean
```

#### Working with PostgreSQL

```bash
# Connect with psql
./pgbox psql

# Connect to specific container by name
./pgbox psql --name my-postgres-dev

# Pass arguments to psql
./pgbox psql -- -c "SELECT version();"

# List available extensions
./pgbox list-extensions

# Search for specific extensions
./pgbox list-extensions | grep vector
```

#### Exporting for Production

```bash
# Export Docker configuration to current directory
./pgbox export

# Export to specific directory
./pgbox export --dir ./my-postgres-setup

# Export with specific extensions
./pgbox export --ext pgvector,hypopg --dir ./prod-setup

# Generated files:
# - Dockerfile: Custom image with extensions
# - docker-compose.yml: Complete Docker Compose setup
# - init.sql: SQL script to create extensions
```

### Examples

#### Development Setup with Vector Search

```bash
# Start PostgreSQL with pgvector for AI/ML applications
./pgbox up --ext pgvector --name vector-dev

# Connect and create vector extension
./pgbox psql --name vector-dev
```

#### Testing with Multiple Extensions

```bash
# Start with common extensions for testing
./pgbox up --ext postgis,pg_trgm,uuid-ossp,hstore

# Run your tests against localhost:5432

# Clean up when done
./pgbox down --volumes
```

#### Export for Docker Compose Project

```bash
# Create a production-ready setup
./pgbox export --ext pgvector,pg_stat_statements --dir ./docker

# Use the generated files in your project
cd ./docker
docker compose up -d
```

### Extension Support

pgbox supports hundreds of PostgreSQL extensions from apt.postgresql.org. Some popular ones:

- **pgvector**: Vector similarity search for AI applications
- **postgis**: Geographic objects and spatial processing
- **hypopg**: Hypothetical indexes for query planning
- **pg_trgm**: Trigram-based text similarity
- **pg_stat_statements**: Query performance tracking
- **uuid-ossp**: UUID generation functions
- **hstore**: Key-value store within PostgreSQL
- **pg_cron**: Job scheduling inside PostgreSQL

View all available extensions:

```bash
./pgbox list-extensions
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

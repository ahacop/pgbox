# pgbox

PostgreSQL-in-Docker with selectable extensions.

## Overview

pgbox is a CLI tool that simplifies running PostgreSQL in Docker with your choice of extensions. It provides an easy way to spin up PostgreSQL instances with specific extensions for development and testing purposes.

### Purpose & Philosophy

pgbox is designed as an **experimentation and prototyping tool** for PostgreSQL extensions. Its primary goal is to help you quickly test different extensions and configurations before committing to them in your project:

- **Quick experimentation**: Spin up a PostgreSQL instance with any combination of extensions in seconds
- **Test before you commit**: Try out extensions locally before adding them to your production setup
- **Export when ready**: Once you've found the right configuration, export it as Docker files for your project
- **Clean, isolated environments**: Each unique extension combination gets its own container, preventing conflicts from incompatible extensions

Think of pgbox as your PostgreSQL sandbox - a place to freely experiment with the vast ecosystem of PostgreSQL extensions without worrying about breaking your development database or dealing with complex installation procedures.

### Key Features

- **200+ Extensions**: Comprehensive support for PostgreSQL extensions from apt.postgresql.org
- **TOML-based Configuration**: Declarative extension specifications with support for complex configurations
- **Smart Configuration Merging**: Automatically handles shared_preload_libraries and PostgreSQL GUCs
- **Export to Docker**: Generate production-ready Docker Compose configurations
- **Multiple PostgreSQL Versions**: Support for PostgreSQL 16 and 17
- **Development-Friendly**: Quick spin-up of PostgreSQL instances with specific extensions

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
# Export Docker configuration to directory
./pgbox export ./my-postgres

# Export with specific version and extensions
./pgbox export ./my-postgres -v 16 --ext pgvector,hypopg

# Export with custom port
./pgbox export ./my-postgres -p 5433

# Export with custom base image
./pgbox export ./my-postgres --base-image postgres:17-alpine

# Generated files:
# - Dockerfile: Custom image with extensions
# - docker-compose.yml: Complete Docker Compose setup with required configurations
# - init.sql: SQL script to create extensions
# - postgresql.conf (if needed): PostgreSQL configuration for extensions requiring preload
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
./pgbox export ./docker --ext pgvector,pg_stat_statements

# Extensions with complex requirements (e.g., pg_cron)
./pgbox export ./docker --ext pg_cron,wal2json

# Use the generated files in your project
cd ./docker
docker compose up -d
```

#### Working with Extensions Requiring Preload

Some extensions like `pg_cron` and `wal2json` require shared_preload_libraries:

```bash
# Export with pg_cron (automatically configures preload and GUCs)
./pgbox export ./cron-setup --ext pg_cron

# The system will:
# - Add pg_cron to shared_preload_libraries
# - Configure cron.database_name and other GUCs
# - Generate proper init.sql with CREATE EXTENSION
```

### Extension Support

pgbox supports 200+ PostgreSQL extensions from apt.postgresql.org. Each extension is defined using a TOML specification that includes:

- Package dependencies
- PostgreSQL configuration requirements (shared_preload_libraries, GUCs)
- SQL initialization commands
- Docker compose hints

#### Popular Extensions

- **pgvector**: Vector similarity search for AI applications
- **postgis**: Geographic objects and spatial processing
- **pg_cron**: Job scheduling inside PostgreSQL (with automatic preload configuration)
- **hypopg**: Hypothetical indexes for query planning
- **pg_stat_statements**: Query performance tracking
- **timescaledb**: Time-series data optimization
- **wal2json**: Logical replication with JSON output
- **pgtap**: Unit testing framework for PostgreSQL
- **uuid-ossp**: UUID generation functions
- **hstore**: Key-value store within PostgreSQL

View all available extensions:

```bash
./pgbox list-extensions
```

#### Extension Configuration System

Extensions are defined in TOML files under `extensions/<name>/<version>.toml`. Complex extensions can specify:

```toml
# Example: pg_cron configuration
[postgresql.conf]
shared_preload_libraries = ["pg_cron"]
"cron.database_name" = "postgres"

[[sql.initdb]]
text = "CREATE EXTENSION IF NOT EXISTS pg_cron;"
```

The system automatically:
- Merges configurations from multiple extensions
- Detects and reports conflicts
- Preserves user customizations in generated files

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

# Update extension catalogs and generate TOML files
make update-extensions

# Generate TOML files from existing JSON data
make generate-toml
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Test export functionality
make export EXTS=pgvector,pg_cron PG_VERSION=17

# Run PostgreSQL with extensions
make run EXTS=pgvector,pg_cron PORT=5432
```

## Architecture

### Extension System

The extension system uses a declarative TOML-based approach:

1. **Extension Specifications** (`extensions/<name>/<version>.toml`): Define package dependencies, PostgreSQL configuration, and SQL initialization
2. **Model Layer** (`internal/model/`): In-memory representations of Docker artifacts
3. **Applier** (`internal/applier/`): Merges extension requirements into models
4. **Renderer** (`internal/render/`): Generates Docker files with anchored blocks for user customizations

### Key Components

- **cmd/**: Command implementations (up, down, psql, export, etc.)
- **internal/config/**: PostgreSQL configuration management
- **internal/container/**: Container lifecycle management
- **internal/docker/**: Docker command wrapper
- **internal/extensions/**: Extension validation and TOML loading
- **pkg/scaffold/**: Template-based Docker artifact generation

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT

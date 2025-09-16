# pgbox

PostgreSQL-in-Docker with extensions.

## Overview

pgbox is a CLI tool that simplifies running PostgreSQL in Docker with your choice of extensions.

### Purpose & Philosophy

pgbox is designed as an **experimentation and prototyping tool** for PostgreSQL extensions. Its primary goal is to help you quickly experiment and test different extensions and configurations before committing to them in your project:

- **Quick experimentation**: Spin up a PostgreSQL instance with any combination of extensions in seconds
- **Test before you commit**: Try out extensions locally before adding them to your production setup
- **Export when ready**: Once you've found the right configuration, export it as Docker files for your project
- **Clean, isolated environments**: Each unique extension combination gets its own container, preventing conflicts from incompatible extensions

Think of pgbox as your PostgreSQL sandbox - a place to freely experiment with the vast ecosystem of PostgreSQL extensions without worrying about breaking your development database or dealing with complex installation procedures.

### Key Features

- **Development-Friendly**: Quick spin-up of PostgreSQL instances with specific extensions
- **Export to Docker**: Generate production-ready Docker Compose configurations
- **200+ Extensions**: Comprehensive support for PostgreSQL extensions from apt.postgresql.org

## Installation

### Download Pre-built Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/ahacop/pgbox/releases).

#### Using the install script

```bash
curl -sSL https://raw.githubusercontent.com/ahacop/pgbox/main/install.sh | sh
```

#### Manual download

1. Download the appropriate archive for your platform from [releases](https://github.com/ahacop/pgbox/releases)
2. Extract and move to your PATH:

```bash
tar -xzf pgbox_*_Linux_x86_64.tar.gz
sudo mv pgbox /usr/local/bin/
```

### Using Go

```bash
go install github.com/ahacop/pgbox@latest
```

### Using Nix Flake

```bash
# Run directly without installation
nix run github:ahacop/pgbox -- --help

# Start with specific extensions
nix run github:ahacop/pgbox -- up --ext pgvector,postgis
```

### Build from Source

```bash
git clone https://github.com/ahacop/pgbox
cd pgbox
make build

# Optional: Install to GOPATH/bin
make install
```

## Usage

### Quick Start

```bash
# Start PostgreSQL with default settings (no extensions)
./pgbox up

# Start PostgreSQL with specific extensions
./pgbox up --ext pgvector,hypopg

# List available extensions
./pgbox list-extensions

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

### Extension Support

pgbox supports 200+ PostgreSQL extensions from apt.postgresql.org. Each extension is defined using a TOML specification that includes:

- Package dependencies
- PostgreSQL configuration requirements (shared_preload_libraries, GUCs)
- SQL initialization commands
- Docker compose hints

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

## Release Process (Maintainers)

### Creating a Release

1. **Ensure all changes are committed and pushed**

   ```bash
   git status
   git push origin main
   ```

2. **Test the release locally**

   ```bash
   # Test the release build without publishing
   make release-snapshot

   # Check generated artifacts
   ls -la dist/
   ```

3. **Create and push a version tag**

   ```bash
   # For a new minor version
   git tag v0.2.0
   git push origin v0.2.0

   # For a patch version
   git tag v0.2.1
   git push origin v0.2.1
   ```

4. **GitHub Actions will automatically**:
   - Run tests
   - Build binaries for Linux and macOS (amd64/arm64)
   - Create a GitHub release with:
     - Pre-built binaries for all platforms
     - SHA256 checksums
     - Auto-generated changelog from commit messages

5. **After release, users can install via**:
   - Pre-built binaries from GitHub releases
   - Install script: `curl -sSL https://raw.githubusercontent.com/ahacop/pgbox/main/install.sh | sh`
   - Go install: `go install github.com/ahacop/pgbox@v0.2.0`

## License

GPL-3.0

# pgbox 🐘📦

**pgbox** is a CLI tool for running PostgreSQL in Docker with your choice of extensions.

It is designed for **experimentation and prototyping**, making it easy to test extensions before using them in a project.

### Why pgbox?

- **Quick setup**: Spin up PostgreSQL with any set of extensions in seconds
- **Easy experimentation**: Test extensions locally before adding them to your stack
- **Export to Docker**: Export Docker Compose files when ready for your project
- **150+ extensions supported**: From [apt.postgresql.org](https://apt.postgresql.org)

Think of pgbox as a PostgreSQL sandbox — a safe place to explore 150+ extensions without manual installs or risking your development database.

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

#### Exporting for your project

```bash
# Export Docker configuration to directory
./pgbox export ./my-postgres

# Export with specific version and extensions
./pgbox export ./my-postgres -v 16 --ext pgvector,hypopg

# Export with custom port
./pgbox export ./my-postgres -p 5433

# Generated files:
# - Dockerfile: Custom image with extensions
# - docker-compose.yml: Complete Docker Compose setup with required configurations
# - init.sql: SQL script to create extensions
# - postgresql.conf (if needed): PostgreSQL configuration for extensions requiring preload
```

## Development

### Prerequisites

- Go 1.24+
- Docker
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

# Test export functionality
make export EXTS=pgvector,pg_cron PG_VERSION=17

# Run PostgreSQL with extensions
make run EXTS=pgvector,pg_cron PORT=5432
```

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

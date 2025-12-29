# ADR-002: PostgreSQL Extension System Architecture (TOML-based)

## Status

Superseded by [ADR-003](003-go-extension-catalog.md)

## Date

2025-01-16

## Context

pgbox needs to manage PostgreSQL extensions across different contexts:

- Quick prototyping with `pgbox up` command
- Production-ready Docker artifact generation with `pgbox export` command
- Support for 200+ extensions from apt.postgresql.org
- Complex extension requirements (shared libraries, GUCs, SQL initialization)

### Current Challenges

1. **Complex Extension Requirements**: Extensions like pg_cron, timescaledb, and wal2json require:
   - Specific shared libraries preloaded at startup
   - Custom PostgreSQL configuration parameters
   - Initialization SQL in specific order

2. **Multi-Version Support**: Need to handle:
   - Different package names across PostgreSQL versions
   - Version-specific configuration requirements

## Decision

Implement a declarative, TOML-based extension specification system with a clear separation of concerns:

### 1. Extension Specifications (TOML Schema)

Each extension is defined by a TOML file containing:

```toml
# Identity and metadata
extension = "pg_cron"          # SQL name for CREATE EXTENSION
display_name = "cron"          # Human-friendly name
package = "postgresql-16-cron" # APT package name
description = "Run periodic jobs in PostgreSQL"

# Docker image mutations
[image]
apt_packages = ["postgresql-16-cron"]

# PostgreSQL configuration
[postgresql.conf]
shared_preload_libraries = ["pg_cron"]
"cron.database_name" = "postgres"  # Custom GUCs

# SQL initialization
[[sql.initdb]]
text = "CREATE EXTENSION IF NOT EXISTS pg_cron;"

[[sql.poststart]]  # For commands requiring running server
text = "SELECT cron.schedule(...);"

# pgbox hints
[pgbox]
needs_restart = true  # Requires server restart after config
compose_env = { CRON_DATABASE = "postgres" }
ports = ["8080:8080"]  # Additional ports to expose
```

### 2. Layered Architecture

The extension system uses a four-layer architecture:

```
┌─────────────────────────────────────┐
│         TOML Specifications         │  ← Declarative extension definitions
│     (extensions/{name}/{ver}.toml)  │
└────────────┬────────────────────────┘
             │ Load & Validate
┌────────────▼────────────────────────┐
│          ExtSpec Loader             │  ← Schema validation & normalization
│     (internal/extspec/spec.go)      │
└────────────┬────────────────────────┘
             │ Transform
┌────────────▼────────────────────────┐
│           Model Layer               │  ← In-memory representations
│        (internal/model/*.go)        │
│  • DockerfileModel                  │
│  • ComposeModel                     │
│  • PGConfModel                      │
│  • InitModel                        │
└────────────┬────────────────────────┘
             │ Apply
┌────────────▼────────────────────────┐
│            Applier                  │  ← Merge specs into models
│    (internal/applier/apply.go)      │  ← Conflict detection
└────────────┬────────────────────────┘
             │ Render
┌────────────▼────────────────────────┐
│           Renderer                  │  ← Generate Docker artifacts
│     (internal/render/*.go)          │  ← Anchored blocks for customization
└─────────────────────────────────────┘
```

### 3. Data Flow

1. **Loading Phase**:
   - Loader reads TOML files from `extensions/{name}/{version}.toml`
   - Falls back to `default.toml` if version-specific not found
   - Validates schema (identifiers, GUCs, ports)
   - Normalizes data (deduplication, sorting)

2. **Model Phase**:
   - Creates empty model instances for each artifact type
   - Models provide type-safe APIs for building configurations
   - Models maintain internal consistency (e.g., unique packages)

3. **Application Phase**:
   - Applier iterates through loaded specs
   - Merges requirements into models
   - Detects and reports conflicts (e.g., conflicting GUC values)
   - Maintains source tracking for debugging

4. **Rendering Phase**:
   - Renderer generates final Docker artifacts
   - Preserves user customizations via anchored blocks
   - Produces deterministic output for reproducibility

### 4. Conflict Resolution

The system detects and reports configuration conflicts:

```go
type Conflict struct {
    Type       string   // "GUC", "Port", etc.
    Key        string   // Conflicting key
    Extensions []string // Extensions involved
    Values     []string // Conflicting values
}
```

When conflicts are detected, the system:

1. Collects all conflicts before failing
2. Reports comprehensive error with all conflicts
3. Suggests resolution strategies

### 5. Extension Discovery and Generation

Extensions are discovered and maintained through:

- File system scanning of `extensions/` directory
- Dynamic loading based on requested PostgreSQL version
- Build-time generation from official PostgreSQL Docker images

#### Build-Time Data Pipeline

The scripts in `scripts/` folder create a data generation pipeline:

1. **Data Collection**: Scripts query official PostgreSQL Docker images to discover available extensions
2. **Mapping Generation**: Build extension name mappings between package names and SQL names
3. **TOML Generation**: Automatically generate TOML specifications from collected data
4. **Manual Enhancement**: TOML files can be manually edited to add configuration requirements

This ensures extension data stays synchronized with official PostgreSQL repositories.

## Implementation

### Key Components

1. **ExtensionSpec** (internal/extspec/spec.go:16-41):
   - Core data structure for extension configuration
   - Nested structure for PostgreSQL configuration
   - Support for multiple package managers

2. **Loader** (internal/extspec/spec.go:113-194):
   - TOML parsing and validation
   - Version-specific file resolution
   - Batch loading for multiple extensions

3. **Applier** (internal/applier/apply.go:13-65):
   - Stateful application of specs to models
   - Conflict detection and tracking
   - Package manager detection

4. **Model Layer** (internal/model/):
   - Type-safe representations of Docker artifacts
   - Builder pattern APIs for incremental construction
   - Validation at model level

5. **Renderer** (internal/render/):
   - Template-based generation
   - Anchored blocks for user customization
   - Deterministic output ordering

## Consequences

### Positive

1. **Declarative Configuration**: Extensions defined in simple, readable TOML
2. **Separation of Concerns**: Clear boundaries between loading, modeling, and rendering
3. **Conflict Detection**: Early detection of incompatible extensions
4. **Extensibility**: Easy to add new configuration options
5. **Multi-Version Support**: Graceful handling of version differences
6. **Production Ready**: Export generates complete Docker configurations

### Negative

1. **Complexity**: Four-layer architecture adds indirection
2. **Manual Enhancement**: Complex extensions need manual TOML editing after generation
3. **Limited Validation**: Can't validate runtime behavior, only configuration
4. **Build Dependencies**: Requires Docker for updating extension catalogs

### Maintenance Burden

- **Low**: Adding new extensions only requires TOML files
- **Medium**: Schema changes require updates across all layers
- **Low**: Updating extension catalogs is automated via `make update-extensions`

## Alternatives Considered

1. **Single JSONL File**:
   - Simpler but less expressive
   - No support for complex configurations
   - Difficult to version control changes

2. **Code-Based Configuration**:
   - More flexible but less accessible
   - Requires recompilation for changes

3. **YAML Instead of TOML**:
   - More familiar to Docker users
   - More complex parsing
   - Indentation-sensitive (error-prone)

4. **Database-Backed Configuration**:
   - Overkill for CLI tool
   - Adds deployment complexity
   - Requires migration strategy

## Future Improvements

1. **Extension Dependency Resolution**: Automatically include required extensions
2. **Conflict Auto-Resolution**: Smart merging of compatible configurations
3. **Extension Marketplace**: Community-contributed extension specifications
4. **Runtime Validation**: Test extension configurations in ephemeral containers
5. **Version Constraints**: Support minimum/maximum PostgreSQL versions in specs
6. **Automated Configuration Discovery**: Analyze extension source to auto-detect requirements

## References

- [TOML Specification](https://toml.io/en/)
- [PostgreSQL Extension Documentation](https://www.postgresql.org/docs/current/extend-extensions.html)
- [Docker Multi-stage Builds](https://docs.docker.com/develop/develop-images/multistage-build/)
- [apt.postgresql.org Repository](https://apt.postgresql.org/)

---

## Supersession Note

This ADR was superseded in December 2025. The TOML-based extension system described here was replaced with a simpler Go-based catalog approach. See [ADR-003](003-go-extension-catalog.md) for the rationale and details of the new architecture.

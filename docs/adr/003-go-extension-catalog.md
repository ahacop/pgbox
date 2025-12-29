# ADR-003: Go-based Extension Catalog

## Status

Accepted

## Date

2025-12-29

## Context

[ADR-002](002-extension-system-architecture.md) described a TOML-based extension specification system with a four-layer architecture (TOML specs → Loader → Model Layer → Applier → Renderer). While this approach was comprehensive, it introduced significant complexity:

1. **Over 300 TOML files** needed maintenance across PostgreSQL versions
2. **Four-layer architecture** added indirection that complicated debugging
3. **Build-time generation** required scripts to keep TOML files synchronized
4. **Schema complexity** made it harder to add new extensions

In practice, most extensions fell into three simple categories:
- Built-in contrib extensions (no configuration needed)
- Simple third-party extensions (just need apt package)
- Complex extensions (need shared_preload_libraries and/or GUCs)

## Decision

Replace the TOML-based system with a single Go map in `internal/extensions/catalog.go`.

### Extension Definition

```go
type Extension struct {
    Package string            // apt package pattern (e.g., "postgresql-{v}-pgvector")
    SQLName string            // CREATE EXTENSION name if different from key
    Preload []string          // shared_preload_libraries entries
    GUCs    map[string]string // PostgreSQL configuration parameters
    InitSQL string            // Custom initialization SQL
}

var Catalog = map[string]Extension{
    // Built-in (empty struct)
    "hstore": {},
    "ltree":  {},

    // Simple third-party
    "pgvector": {Package: "postgresql-{v}-pgvector", SQLName: "vector"},
    "hypopg":   {Package: "postgresql-{v}-hypopg"},

    // Complex
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

### Helper Functions

The catalog provides type-safe helper functions:
- `Get(name)` - lookup extension configuration
- `GetPackage(name, version)` - get apt package with version substituted
- `GetSQLName(name)` - get SQL name for CREATE EXTENSION
- `GetInitSQL(name)` - get initialization SQL
- `ValidateExtensions(names)` - validate extensions exist
- `ListExtensions()` - list all extension names
- `GetPackages(names, version)` - get all apt packages needed
- `GetPreloadLibraries(names)` - get all shared_preload_libraries
- `GetGUCs(names)` - get all GUCs with conflict detection

### Architecture Simplification

The new architecture has two layers:

```
┌─────────────────────────────────────┐
│         Extension Catalog           │  ← Single Go map with helpers
│  (internal/extensions/catalog.go)   │
└────────────┬────────────────────────┘
             │ Direct access
┌────────────▼────────────────────────┐
│           Orchestrators             │  ← Business logic uses helpers
│     (internal/orchestrator/*.go)    │
│                                     │
│           Renderers                 │  ← Generate Docker artifacts
│       (internal/render/*.go)        │
└─────────────────────────────────────┘
```

## Consequences

### Positive

1. **Simplicity**: Single file with ~350 lines replaces 300+ TOML files
2. **Type Safety**: Go compiler catches errors; no runtime TOML parsing
3. **Maintainability**: Adding extensions is a one-line change
4. **Testability**: Helper functions are easily unit tested
5. **Performance**: No file I/O or parsing at runtime
6. **Refactoring**: IDE support for renaming, finding references
7. **Conflict Detection**: `GetGUCs()` detects conflicting settings at call time

### Negative

1. **Requires Recompilation**: Adding extensions requires rebuilding the binary
2. **Less Declarative**: Configuration is in code rather than data files
3. **Version Coupling**: Extension configs tied to pgbox releases

### Trade-offs Accepted

The trade-offs are acceptable because:
- Extension configurations rarely change
- The catalog is already comprehensive (150+ extensions)
- Users needing custom extensions can fork or use `pgbox export` then modify

## Alternatives Reconsidered

1. **Keep TOML System**: Rejected due to maintenance burden
2. **JSON/YAML Config**: Same complexity as TOML without benefits
3. **Embedded TOML**: Would preserve declarative nature but add parsing complexity
4. **Database/API**: Overkill for a CLI tool

## Implementation Notes

The migration was completed in commit `abeff01`:
- Deleted `extensions/` directory (300+ TOML files)
- Deleted `internal/extspec/` and `internal/applier/` packages
- Created `internal/extensions/catalog.go`
- Updated orchestrators to use catalog helpers directly

## References

- [Go Maps](https://go.dev/blog/maps)
- [ADR-002: Original TOML Architecture](002-extension-system-architecture.md)

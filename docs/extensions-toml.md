# PostgreSQL Extension TOML Specification

## Overview

pgbox uses TOML files to declaratively define PostgreSQL extensions and their requirements. Each extension has a TOML file that specifies:

- Package dependencies
- PostgreSQL configuration requirements
- SQL initialization commands
- Docker compose hints

## File Structure

Extension TOML files are located at: `extensions/<extension-name>/<pg-major>.toml`

Example: `extensions/pgvector/17.toml` for pgvector on PostgreSQL 17

## Schema Definition

### Root Fields

| Field          | Type   | Required | Description                                          |
| -------------- | ------ | -------- | ---------------------------------------------------- |
| `extension`    | string | Yes      | The SQL name used in CREATE EXTENSION                |
| `display_name` | string | No       | Human-friendly display name                          |
| `package`      | string | No       | Full apt package name (e.g., postgresql-17-pgvector) |
| `description`  | string | No       | Brief description of the extension                   |
| `min_pg`       | string | No       | Minimum PostgreSQL version (semver)                  |
| `max_pg`       | string | No       | Maximum PostgreSQL version (semver)                  |

### [image] Section

Defines packages to install in the Docker image.

| Field          | Type     | Description                               |
| -------------- | -------- | ----------------------------------------- |
| `apt_packages` | string[] | Debian/Ubuntu packages to install via apt |

### [postgresql.conf] Section

PostgreSQL configuration parameters.

| Field                      | Type     | Description                                |
| -------------------------- | -------- | ------------------------------------------ |
| `shared_preload_libraries` | string[] | Libraries to preload at server start       |
| `*`                        | string   | Any other PostgreSQL GUC as key-value pair |

Example:

```toml
[postgresql.conf]
shared_preload_libraries = ["pg_cron"]
"cron.database_name" = "postgres"
"wal_level" = "logical"
```

### [[sql.initdb]] Array

SQL commands to execute during database initialization (via docker-entrypoint-initdb.d).

| Field  | Type   | Description            |
| ------ | ------ | ---------------------- |
| `text` | string | SQL command to execute |

### [[sql.poststart]] Array

SQL commands to execute after the server is fully started (optional).

| Field  | Type   | Description            |
| ------ | ------ | ---------------------- |
| `text` | string | SQL command to execute |

### [pgbox] Section

Hints for the pgbox engine.

| Field           | Type     | Description                                                   |
| --------------- | -------- | ------------------------------------------------------------- |
| `needs_restart` | boolean  | True if changes require server restart                        |
| `compose_env`   | map      | Environment variables for docker-compose                      |
| `ports`         | string[] | Additional ports to expose (format: "host:container[/proto]") |

## Merge Strategy

When multiple extensions are selected, pgbox merges their requirements:

### Package Merging

- All packages are collected into a unique, sorted list
- Duplicates are automatically removed

### Shared Preload Libraries

- Libraries from all extensions are merged into a unique, sorted list
- Order is deterministic (alphabetical)

### GUC Merging

- If multiple extensions set the same GUC to different values: **ERROR**
- If multiple extensions set the same GUC to the same value: OK
- Error message includes which extensions conflict on which GUC

### SQL Fragment Merging

- Each fragment is hashed (SHA-256) based on normalized content
- Duplicate fragments (same hash) are skipped
- Fragments are wrapped with anchored comments for tracking

### Compose Hints

- Environment variables: last-writer-wins with warning
- Ports: all ports are included

## Examples

### Minimal Extension (Auto-generated)

```toml
# extensions/btree_gin/17.toml
extension = "btree_gin"
description = "support for indexing common datatypes in GIN"

[[sql.initdb]]
text = "CREATE EXTENSION IF NOT EXISTS btree_gin;"
```

### Simple Package Extension

```toml
# extensions/pgvector/17.toml
extension = "vector"
display_name = "pgvector"
package = "postgresql-17-pgvector"
description = "vector data type and ivfflat and hnsw access methods"

[image]
apt_packages = ["postgresql-17-pgvector"]

[[sql.initdb]]
text = "CREATE EXTENSION IF NOT EXISTS vector;"
```

### Complex Extension with Preload

```toml
# extensions/pg_cron/17.toml
extension = "pg_cron"
display_name = "pg_cron"
package = "postgresql-17-cron"
description = "Run periodic jobs in PostgreSQL"
min_pg = "14"

[image]
apt_packages = ["postgresql-17-cron"]

[postgresql.conf]
shared_preload_libraries = ["pg_cron"]
"cron.database_name" = "postgres"
"cron.max_running_jobs" = "5"

[[sql.initdb]]
text = "CREATE EXTENSION IF NOT EXISTS pg_cron;"

[[sql.initdb]]
text = "GRANT USAGE ON SCHEMA cron TO postgres;"

[pgbox]
needs_restart = true
```

### Logical Replication Extension

```toml
# extensions/wal2json/17.toml
extension = "wal2json"
display_name = "wal2json"
package = "postgresql-17-wal2json"
description = "PostgreSQL logical decoding JSON output plugin"

[image]
apt_packages = ["postgresql-17-wal2json"]

[postgresql.conf]
shared_preload_libraries = ["wal2json"]
"wal_level" = "logical"
"max_replication_slots" = "10"
"max_wal_senders" = "10"

# wal2json is an output plugin, no CREATE EXTENSION needed
[[sql.initdb]]
text = "-- wal2json logical decoding plugin configured"

[pgbox]
needs_restart = true
```

## Validation Rules

1. **Extension name**: Must be valid PostgreSQL identifier
2. **Package names**: Must follow apt naming conventions
3. **GUC keys**: Letters, digits, underscore, dot only (no spaces)
4. **Semver ranges**: Valid semantic version strings
5. **SQL text**: Non-empty strings
6. **Ports**: Valid port format "host:container" or "host:container/proto"

## File Generation

Base TOML files are auto-generated from pgbox's extension catalog using:

```bash
make generate-toml
```

This creates minimal TOML files with:

- Basic metadata from the catalog
- Package installation (for non-builtin extensions)
- Simple CREATE EXTENSION statement

Complex requirements (preload libraries, GUCs, etc.) must be added manually.

## Anchored Blocks in Output

Generated files use anchored blocks to preserve user customizations:

### Dockerfile

```dockerfile
# pgbox: BEGIN apt
RUN apt-get update && apt-get install -y \
    postgresql-17-pgvector \
    postgresql-17-cron
# pgbox: END apt
```

### docker-compose.yml

```yaml
# pgbox: BEGIN
services:
  db:
    environment:
      POSTGRES_USER: postgres
    # ... managed configuration ...
# pgbox: END
```

### init.sql

```sql
-- pgbox: begin pgvector sha256=abc123...
CREATE EXTENSION IF NOT EXISTS vector;
-- pgbox: end pgvector

-- pgbox: begin pg_cron sha256=def456...
CREATE EXTENSION IF NOT EXISTS pg_cron;
-- pgbox: end pg_cron
```

Content outside these blocks is preserved when regenerating files.

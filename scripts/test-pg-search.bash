#!/usr/bin/env bash
# TDD test harness for pg_search extension via .deb installation
# This test should FAIL until we implement .deb support
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PGBOX="$PROJECT_ROOT/pgbox"

# Cleanup function
cleanup() {
    local exit_code=$?
    echo "Cleaning up..."
    rm -rf "$EXPORT_DIR" 2>/dev/null || true
    exit $exit_code
}
trap cleanup EXIT

# Create temp directory for export
EXPORT_DIR=$(mktemp -d)
echo "Export directory: $EXPORT_DIR"

# Step 1: Build pgbox if needed
echo "Building pgbox..."
cd "$PROJECT_ROOT"
make build

# Step 2: Export Docker artifacts with pg_search extension
echo "Exporting Docker artifacts with pg_search..."
"$PGBOX" export "$EXPORT_DIR" --ext pg_search -v 17

# Step 3: Build the Docker image
echo "Building Docker image..."
docker build -t pgbox-pg-search-test:latest "$EXPORT_DIR"

# Step 4: Run one-liner test - create extension and run a query
# pg_search requires shared_preload_libraries for PG < 17, but we're using PG17 so it should work
echo "Testing pg_search extension..."
RESULT=$(docker run --rm \
    -e POSTGRES_PASSWORD=postgres \
    pgbox-pg-search-test:latest \
    postgres -c 'shared_preload_libraries=pg_search' \
    -c "SELECT 'pg_search test'" \
    2>&1 || true)

# Actually, the above won't work because postgres command doesn't take SQL directly
# We need to start postgres and then run psql against it
# Let's use docker run with a proper test

echo "Running proper pg_search test..."
TEST_RESULT=$(docker run --rm \
    -e POSTGRES_PASSWORD=postgres \
    -e POSTGRES_HOST_AUTH_METHOD=trust \
    pgbox-pg-search-test:latest \
    bash -c '
        # Start postgres in background
        docker-entrypoint.sh postgres -c shared_preload_libraries=pg_search &
        PG_PID=$!

        # Wait for postgres to be ready
        for i in {1..30}; do
            if pg_isready -U postgres -q 2>/dev/null; then
                break
            fi
            sleep 1
        done

        # Run test queries
        psql -U postgres -d postgres <<EOF
-- Create the extension
CREATE EXTENSION IF NOT EXISTS pg_search;

-- Create a test table
CREATE TABLE test_docs (
    id SERIAL PRIMARY KEY,
    title TEXT,
    body TEXT
);

-- Insert test data
INSERT INTO test_docs (title, body) VALUES
    ('"'"'PostgreSQL Full Text'"'"', '"'"'PostgreSQL has built-in full text search'"'"'),
    ('"'"'BM25 Algorithm'"'"', '"'"'pg_search uses the BM25 ranking algorithm'"'"'),
    ('"'"'Tantivy Engine'"'"', '"'"'Built on Rust Tantivy search engine'"'"');

-- Create BM25 index
CREATE INDEX search_idx ON test_docs
USING bm25 (id, title, body)
WITH (key_field='"'"'id'"'"');

-- Test search query
SELECT title, body FROM test_docs WHERE title ||| '"'"'PostgreSQL'"'"' LIMIT 1;

-- Verify extension is loaded
SELECT extname, extversion FROM pg_extension WHERE extname = '"'"'pg_search'"'"';
EOF

        # Cleanup
        kill $PG_PID 2>/dev/null || true
    ' 2>&1)

echo "Test output:"
echo "$TEST_RESULT"

# Check for success indicators - look for the extension in pg_extension output
if echo "$TEST_RESULT" | grep -q "CREATE EXTENSION" && echo "$TEST_RESULT" | grep -q "pg_search" && ! echo "$TEST_RESULT" | grep -q "FATAL"; then
    echo ""
    echo "SUCCESS: pg_search extension is working!"
    exit 0
else
    echo ""
    echo "FAILURE: pg_search extension test failed"
    echo "Check the output above for errors"
    exit 1
fi

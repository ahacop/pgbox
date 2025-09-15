#!/usr/bin/env bash
set -euo pipefail

# Test script to validate package-to-extension mapping with multiple packages
# Usage: ./test-extension-mapping.bash [pg-major]

MAJOR="${1:-17}"

# Test packages with known extensions
TEST_PACKAGES=(
	"pgvector:vector"
	"hypopg:hypopg"
	"pg-stat-statements:pg_stat_statements"
	"postgis-3:postgis,address_standardizer,postgis_raster,postgis_topology"
)

echo "Testing extension mapping for PostgreSQL $MAJOR"
echo "============================================"

# Start a shared container for reuse mode testing
echo "Starting shared container..."
IMG="postgres:${MAJOR}"
CID=$(docker run -d \
	-e POSTGRES_PASSWORD=test \
	-e POSTGRES_HOST_AUTH_METHOD=trust \
	"$IMG" 2>/dev/null)

# Wait for PostgreSQL to be ready
for i in {1..30}; do
	if docker exec "$CID" pg_isready -U postgres >/dev/null 2>&1; then
		break
	fi
	sleep 1
done

# Update apt cache once
echo "Updating package cache..."
docker exec "$CID" apt-get update -qq >/dev/null 2>&1

# Test each package
echo ""
echo "Testing packages:"
for entry in "${TEST_PACKAGES[@]}"; do
	PKG_NAME="${entry%%:*}"
	EXPECTED="${entry#*:}"

	echo ""
	echo "  Package: $PKG_NAME"
	echo "  Expected extensions: $EXPECTED"

	# Get actual extensions
	RESULT=$(./scripts/test-single-package-extensions.bash "$CID" "$PKG_NAME" "$MAJOR" 2>/dev/null || echo "[]")

	if [ "$RESULT" = "[]" ]; then
		echo "  Actual: (none found)"
		echo "  Status: ✗ Failed to find extensions"
	else
		# Convert JSON array to comma-separated list for display
		ACTUAL=$(echo "$RESULT" | jq -r 'join(",")')
		echo "  Actual: $ACTUAL"

		# Check if all expected extensions are present
		IFS=',' read -ra EXPECTED_ARRAY <<< "$EXPECTED"
		ALL_FOUND=true
		for exp in "${EXPECTED_ARRAY[@]}"; do
			if ! echo "$RESULT" | jq -e --arg ext "$exp" 'index($ext)' >/dev/null 2>&1; then
				echo "  Status: ✗ Missing expected extension: $exp"
				ALL_FOUND=false
			fi
		done

		if [ "$ALL_FOUND" = true ]; then
			echo "  Status: ✓ All expected extensions found"
		fi
	fi
done

# Test that uninstall/reinstall works (clean slate)
echo ""
echo "Testing clean slate after uninstall:"
echo "  Installing pgvector..."
FIRST=$(./scripts/test-single-package-extensions.bash "$CID" "pgvector" "$MAJOR" 2>/dev/null)
echo "  First install: $FIRST"

echo "  Installing pgvector again (should get same result)..."
SECOND=$(./scripts/test-single-package-extensions.bash "$CID" "pgvector" "$MAJOR" 2>/dev/null)
echo "  Second install: $SECOND"

if [ "$FIRST" = "$SECOND" ]; then
	echo "  Status: ✓ Clean slate confirmed"
else
	echo "  Status: ✗ Different results on reinstall"
fi

# Clean up
echo ""
echo "Cleaning up..."
docker stop "$CID" >/dev/null 2>&1
docker rm "$CID" >/dev/null 2>&1

echo ""
echo "Test complete!"
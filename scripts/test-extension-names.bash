#!/usr/bin/env bash
set -euo pipefail

# Test script to validate extension name discovery for specific packages
# Usage: ./test-extension-names.bash [package-name] [pg-major]

PKG_NAME="${1:-pgvector}"
MAJOR="${2:-17}"

echo "Testing extension discovery for package: $PKG_NAME (PostgreSQL $MAJOR)"
echo "============================================"

# Test in isolated mode (creates its own container)
echo ""
echo "Running in ISOLATED mode (fresh container):"
ISOLATED_RESULT=$(./scripts/test-single-package-extensions.bash isolated "$PKG_NAME" "$MAJOR")
echo "  Extensions found: $ISOLATED_RESULT"

# Test in reuse mode (uses shared container)
echo ""
echo "Running in REUSE mode (shared container):"
echo "  Starting container..."
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

# Update apt cache
docker exec "$CID" apt-get update -qq >/dev/null 2>&1

# Test the package
REUSE_RESULT=$(./scripts/test-single-package-extensions.bash "$CID" "$PKG_NAME" "$MAJOR")
echo "  Extensions found: $REUSE_RESULT"

# Clean up
docker stop "$CID" >/dev/null 2>&1
docker rm "$CID" >/dev/null 2>&1

# Compare results
echo ""
echo "Comparison:"
echo "  Isolated mode: $ISOLATED_RESULT"
echo "  Reuse mode:    $REUSE_RESULT"

if [ "$ISOLATED_RESULT" = "$REUSE_RESULT" ]; then
	echo "  ✓ Results match!"
else
	echo "  ✗ Results differ!"
	exit 1
fi
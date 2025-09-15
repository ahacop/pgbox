#!/usr/bin/env bash
set -euo pipefail

# Test a single package to discover what PostgreSQL extensions it provides
# This tests in ISOLATION by using a fresh container for each package
# Usage: ./test-single-package-extensions.bash <unused> <package-name> <pg-major>

# First parameter is no longer used but kept for compatibility
_UNUSED="${1:-}"
PKG_NAME="${2:-}"
MAJOR="${3:-}"

if [ "$PKG_NAME" = "" ] || [ "$MAJOR" = "" ]; then
	echo "Usage: $0 <unused> <package-name> <pg-major>"
	echo "This script tests each package in isolation"
	exit 1
fi

PKG="postgresql-${MAJOR}-${PKG_NAME}"
IMG="postgres:${MAJOR}"

# Start a fresh container for this package
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

# Get baseline extensions before install
BEFORE=$(docker exec "$CID" psql -U postgres -t -A -c \
	"SELECT name FROM pg_available_extensions ORDER BY name;" 2>/dev/null)

# Install package
if docker exec "$CID" apt-get install -y "$PKG" >/dev/null 2>&1; then
	# Get extensions after install
	AFTER=$(docker exec "$CID" psql -U postgres -t -A -c \
		"SELECT name FROM pg_available_extensions ORDER BY name;" 2>/dev/null)

	# Find new extensions
	NEW_EXTS=$(comm -13 <(echo "$BEFORE" | sort) <(echo "$AFTER" | sort))

	if [ "$NEW_EXTS" != "" ]; then
		# Output as JSON array
		echo "$NEW_EXTS" | jq -R -s -c 'split("\n") | map(select(length > 0))'
	else
		echo "[]"
	fi
else
	# Installation failed
	echo "[]"
fi

# Clean up container
docker stop "$CID" >/dev/null 2>&1
docker rm "$CID" >/dev/null 2>&1

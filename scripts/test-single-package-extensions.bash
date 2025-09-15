#!/usr/bin/env bash
set -euo pipefail

# Test a single package to discover what PostgreSQL extensions it provides
# Can work in two modes:
# 1. Reuse mode: Use an existing container (pass container ID)
# 2. Isolated mode: Create a fresh container (pass "isolated")
# Usage: ./test-single-package-extensions.bash <container-id|isolated> <package-name> <pg-major>

CID_OR_MODE="${1:-}"
PKG_NAME="${2:-}"
MAJOR="${3:-}"

if [ "$CID_OR_MODE" = "" ] || [ "$PKG_NAME" = "" ] || [ "$MAJOR" = "" ]; then
	echo "Usage: $0 <container-id|isolated> <package-name> <pg-major>" >&2
	echo "  container-id: ID of existing container to reuse" >&2
	echo "  isolated: Create a fresh container for this test" >&2
	exit 1
fi

PKG="postgresql-${MAJOR}-${PKG_NAME}"
CLEANUP_CONTAINER=false

# Determine mode and set container ID
if [ "$CID_OR_MODE" = "isolated" ]; then
	# Create a fresh container for isolated testing
	IMG="postgres:${MAJOR}"
	CID=$(docker run -d \
		-e POSTGRES_PASSWORD=test \
		-e POSTGRES_HOST_AUTH_METHOD=trust \
		"$IMG" 2>/dev/null)
	CLEANUP_CONTAINER=true

	# Wait for PostgreSQL to be ready
	for i in {1..30}; do
		if docker exec "$CID" pg_isready -U postgres >/dev/null 2>&1; then
			break
		fi
		sleep 1
	done

	# Update apt cache
	docker exec "$CID" apt-get update -qq >/dev/null 2>&1
else
	# Use existing container
	CID="$CID_OR_MODE"
fi

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

	# Uninstall the package to restore clean state (only in reuse mode)
	if [ "$CLEANUP_CONTAINER" = false ]; then
		# Remove the package and auto-remove dependencies
		docker exec "$CID" apt-get remove -y "$PKG" >/dev/null 2>&1
		docker exec "$CID" apt-get autoremove -y >/dev/null 2>&1

		# Restart PostgreSQL to ensure clean state
		docker exec "$CID" pg_ctl -D /var/lib/postgresql/data restart -w >/dev/null 2>&1 || true
	fi

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

# Clean up container if we created it
if [ "$CLEANUP_CONTAINER" = true ]; then
	docker stop "$CID" >/dev/null 2>&1
	docker rm "$CID" >/dev/null 2>&1
fi
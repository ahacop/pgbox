#!/usr/bin/env bash
set -euo pipefail

# Discover PostgreSQL extensions from apt.postgresql.org
# Useful when adding support for a new PostgreSQL major version
#
# Usage:
#   ./scripts/discover-extensions.bash 18         # List packages with their SQL names
#   ./scripts/discover-extensions.bash 18 --quick # Just list package names (no SQL discovery)

MAJOR="${1:-}"
QUICK="${2:-}"

if [ -z "$MAJOR" ]; then
	echo "Usage: $0 <pg-major> [--quick]"
	echo "  $0 18         # List packages with SQL extension names (slow, ~5min)"
	echo "  $0 18 --quick # Just list package names (fast)"
	exit 1
fi

IMG="postgres:${MAJOR}"

if [ "$QUICK" = "--quick" ]; then
	echo "Fetching postgresql-${MAJOR}-* packages..."
	docker run --rm "$IMG" bash -c "
		apt-get update -qq 2>/dev/null
		apt-cache search postgresql-${MAJOR}- 2>/dev/null | grep -v dbgsym | sort
	"
	exit 0
fi

echo "Discovering extensions for PostgreSQL ${MAJOR} (this takes a few minutes)..."
docker run --rm -e POSTGRES_HOST_AUTH_METHOD=trust "$IMG" bash -c "
	# Start PostgreSQL in background
	docker-entrypoint.sh postgres &>/dev/null &

	# Wait for ready
	for i in {1..30}; do
		pg_isready -U postgres &>/dev/null && break
		sleep 1
	done

	# Get baseline extensions
	BASELINE=\$(psql -U postgres -Atc 'SELECT name FROM pg_available_extensions' | sort)

	# Update apt
	apt-get update -qq &>/dev/null

	# Get all packages
	PACKAGES=\$(apt-cache search postgresql-${MAJOR}- 2>/dev/null | grep -v dbgsym | awk '{print \$1}' | sort)

	for PKG in \$PACKAGES; do
		SHORT=\${PKG#postgresql-${MAJOR}-}

		# Install
		apt-get install -y \$PKG &>/dev/null 2>&1 || continue

		# Find new extensions
		AFTER=\$(psql -U postgres -Atc 'SELECT name FROM pg_available_extensions' | sort)
		NEW=\$(comm -13 <(echo \"\$BASELINE\") <(echo \"\$AFTER\") | tr '\n' ',' | sed 's/,\$//')

		if [ -n \"\$NEW\" ]; then
			printf '%-40s -> %s\n' \"\$SHORT\" \"\$NEW\"
		fi

		# Update baseline
		BASELINE=\$AFTER
	done
"

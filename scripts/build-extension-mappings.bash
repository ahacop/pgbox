#!/usr/bin/env bash
set -euo pipefail

# Generate package-to-extension name mappings
# This discovers the actual PostgreSQL extension names for each apt package

OUT_DIR="${1:-pgbox-data}"
MAJOR="${2:-}"

generate_mappings() {
  local major="$1"
  local img="postgres:${major}-bookworm"
  local out_file="${OUT_DIR}/extension-mappings-pg${major}.json"

  echo "Generating extension mappings for PostgreSQL ${major}..."

  # Get all packages from our catalog
  local packages=$(jq -r '.entries[] | select(.pkg) | .name' "${OUT_DIR}/apt-pgdg/pg${major}.json")

  # Start a container
  echo "  Starting PostgreSQL container..."
  local cid=$(docker run -d \
    -e POSTGRES_PASSWORD=test \
    -e POSTGRES_HOST_AUTH_METHOD=trust \
    "$img" 2>/dev/null)

  # Wait for PostgreSQL to be ready
  for i in {1..30}; do
    if docker exec "$cid" pg_isready -U postgres >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  # Update apt cache once
  docker exec "$cid" apt-get update -qq >/dev/null 2>&1

  # Initialize mappings object
  local mappings="{}"

  # Test each package
  echo "  Testing packages..."
  for pkg_name in $packages; do
    local pkg="postgresql-${major}-${pkg_name}"
    echo -n "    ${pkg_name}... "

    # Get baseline extensions
    local before=$(docker exec "$cid" psql -U postgres -t -A -c \
      "SELECT name FROM pg_available_extensions ORDER BY name;" 2>/dev/null)

    # Install package
    if docker exec "$cid" apt-get install -y "$pkg" >/dev/null 2>&1; then
      # Get extensions after install
      local after=$(docker exec "$cid" psql -U postgres -t -A -c \
        "SELECT name FROM pg_available_extensions ORDER BY name;" 2>/dev/null)

      # Find new extensions
      local new_exts=$(comm -13 <(echo "$before" | sort) <(echo "$after" | sort) | jq -R -s -c 'split("\n") | map(select(length > 0))')

      if [ "$new_exts" != "[]" ]; then
        # Add to mappings
        mappings=$(echo "$mappings" | jq --arg key "$pkg_name" --argjson val "$new_exts" '. + {($key): $val}')
        echo "found $(echo "$new_exts" | jq -r 'length') extension(s)"
      else
        echo "no new extensions"
      fi
    else
      echo "install failed"
    fi
  done

  # Clean up container
  docker stop "$cid" >/dev/null 2>&1
  docker rm "$cid" >/dev/null 2>&1

  # Write mapping file
  jq -n \
    --arg date "$(date -u +%FT%TZ)" \
    --arg source "$img apt packages" \
    --arg major "$major" \
    --argjson mappings "$mappings" \
    '{
      generated_at: $date,
      source: $source,
      pg_major: ($major | tonumber),
      mappings: $mappings
    }' > "$out_file"

  echo "  Mappings written to $out_file"
}

# Process requested version(s)
if [ -n "$MAJOR" ]; then
  generate_mappings "$MAJOR"
else
  # Generate for both versions
  generate_mappings 16
  generate_mappings 17
fi

echo "Extension mapping generation complete!"
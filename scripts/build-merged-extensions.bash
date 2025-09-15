#!/usr/bin/env bash
set -euo pipefail

# Merge all extension data sources into a single JSONL file
# Each line contains complete information about an extension

DATA_DIR="${1:-pgbox-data}"
OUT_FILE="${DATA_DIR}/extensions.jsonl"

# Function to process extensions for a specific PostgreSQL version
process_pg_version() {
  local major="$1"

  echo "Processing PostgreSQL ${major} extensions..." >&2

  # Load data files
  local builtin_file="${DATA_DIR}/builtin/pg${major}.json"
  local apt_file="${DATA_DIR}/apt-pgdg/pg${major}.json"
  local mappings_file="${DATA_DIR}/extension-mappings-pg${major}.json"

  # Check if mappings exist, otherwise use empty mappings
  local mappings="{}"
  if [ -f "$mappings_file" ]; then
    mappings=$(jq -c '.mappings' "$mappings_file")
  else
    echo "  Warning: No mappings file found at $mappings_file" >&2
  fi

  # Process builtin extensions
  if [ -f "$builtin_file" ]; then
    jq -c --arg major "$major" '.entries[] |
      {
        pg_major: ($major | tonumber),
        name: .name,
        type: "builtin",
        package: null,
        sql_name: .name,
        description: .description
      }' "$builtin_file"
  fi

  # Process apt package extensions with mappings
  if [ -f "$apt_file" ]; then
    # For each package, output an entry for each SQL extension it provides
    jq -c --arg major "$major" --argjson mappings "$mappings" '
      .entries[] |
      select(.pkg) |
      .name as $pkg_name |
      .pkg as $pkg_full |
      .description as $desc |
      ($mappings[$pkg_name] // [$pkg_name]) as $ext_names |
      $ext_names[] as $ext_name |
      {
        pg_major: ($major | tonumber),
        name: $pkg_name,
        type: "package",
        package: $pkg_full,
        sql_name: $ext_name,
        description: $desc
      }' "$apt_file"
  fi
}

# Clear output file
> "$OUT_FILE"

# Process both PostgreSQL versions
for major in 16 17; do
  process_pg_version "$major" >> "$OUT_FILE"
done

# Copy to internal/extensions for embedding
cp "$OUT_FILE" internal/extensions/extensions.jsonl

# Count and report
total_lines=$(wc -l < "$OUT_FILE")
echo ""
echo "Merged extension data written to ${OUT_FILE}"
echo "Also copied to internal/extensions/extensions.jsonl for embedding"
echo "Total entries: ${total_lines}"

# Show statistics
echo ""
echo "Statistics:"
echo "  Builtin extensions: $(grep '"type":"builtin"' "$OUT_FILE" | wc -l)"
echo "  Package extensions: $(grep '"type":"package"' "$OUT_FILE" | wc -l)"
echo ""
echo "Sample entries:"
head -5 "$OUT_FILE" | jq '.'
#!/usr/bin/env bash
set -euo pipefail

# gen-apt-bookworm-pgdg.sh
OUT_DIR="${1:-pgbox-data/apt-pgdg}"
mkdir -p "$OUT_DIR"

gen_for_major() {
  local major="$1"
  local img="postgres:${major}"

  # Do the work using the official postgres image which already has apt repos configured
  mapfile -t pkg_data < <(
    docker run --rm "$img" bash -c "
apt-get update -qq 2>/dev/null
apt-cache search postgresql 2>/dev/null | grep -E '^postgresql-$major-' | grep -v dbgsym | awk '{pkg=\$1; \$1=\"\"; desc=\$0; gsub(/^ /, \"\", desc); print pkg \"|\" desc}' | sort
"
  )

  {
    echo '{'
    echo "  \"generated_at\": \"$(date -u +%FT%TZ)\","
    echo "  \"source\": \"${img} apt packages\","
    echo "  \"pg_major\": ${major},"
    echo '  "entries": ['
    for entry in "${pkg_data[@]}"; do
      IFS='|' read -r pkg desc <<<"$entry"
      n="${pkg#postgresql-"${major}"-}"
      echo "    {\"name\":\"${n}\",\"pkg\":\"${pkg}\",\"description\":\"${desc}\"},"
    done | sed '$ s/},$/}/'
    echo '  ]'
    echo '}'
  } >"${OUT_DIR}/pg${major}.json"
}

gen_for_major 16
gen_for_major 17

echo "APT catalogs written to ${OUT_DIR}/pg16.json and pg17.json"

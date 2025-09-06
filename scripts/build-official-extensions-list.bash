#!/usr/bin/env bash
set -euo pipefail

# gen-builtin.sh
OUT_DIR="${1:-pgbox-data/builtin}"
mkdir -p "$OUT_DIR"

gen_for_major() {
  local major="$1"
  local img="postgres:${major}-bookworm"

  # run a temporary server, query pg_available_extensions
  local cid
  cid="$(docker run -d -e POSTGRES_PASSWORD=pass -p 0:5432 "$img")"
  trap 'docker rm -f "$cid" >/dev/null 2>&1 || true' EXIT

  # wait for readiness
  for _ in {1..60}; do
    if docker exec "$cid" pg_isready -U postgres >/dev/null 2>&1; then break; fi
    sleep 1
  done

  # get extension names and descriptions (what CREATE EXTENSION can see right now)
  local extensions
  extensions="$(docker exec -e PGPASSWORD=pass "$cid" \
    psql -h localhost -U postgres -Atc \
    "SELECT name || '|' || COALESCE(comment, '') FROM pg_available_extensions ORDER BY 1")"

  # write JSON
  {
    echo '{'
    echo "  \"generated_at\": \"$(date -u +%FT%TZ)\","
    echo "  \"source\": \"${img} pg_available_extensions\","
    echo "  \"pg_major\": ${major},"
    echo '  "entries": ['
    awk -F'|' 'NF{printf "    {\"name\":\"%s\",\"kind\":\"builtin\",\"description\":\"%s\"},\n",$1,$2}' <<<"$extensions" | sed '$ s/},$/}/'
    echo '  ]'
    echo '}'
  } >"${OUT_DIR}/pg${major}.json"

  docker rm -f "$cid" >/dev/null
  trap - EXIT
}

gen_for_major 16
gen_for_major 17

echo "Builtin catalogs written to ${OUT_DIR}/pg16.json and pg17.json"

#!/usr/bin/env bash
# Apply docker/postgres/migrations in lexicographic order (001, 002, …).
# Use for CI, local Postgres without Docker init, or after creating an empty database.
# Requires: psql (postgresql-client).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS="${ROOT}/docker/postgres/migrations"

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-app}"

export PGPASSWORD="${POSTGRES_PASSWORD}"

shopt -s nullglob
files=("${MIGRATIONS}"/*.sql)
shopt -u nullglob

if [[ ${#files[@]} -eq 0 ]]; then
	echo "No .sql files in ${MIGRATIONS}" >&2
	exit 1
fi

IFS=$'\n' sorted=($(sort <<<"${files[*]}"))
unset IFS

for f in "${sorted[@]}"; do
	echo "Applying $(basename "$f")"
	psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
		-v ON_ERROR_STOP=1 -f "$f"
done

echo "Migrations applied successfully."

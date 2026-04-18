#!/usr/bin/env bash
# Apply docker/postgres/migrations in lexicographic order (001, 002, …).
# Skips files already recorded in schema_migrations so re-runs are safe on existing DBs.
# Requires: psql (postgresql-client).
#
# Connection: defaults target localhost. Docker Compose maps Postgres to host ports 5432 and 5433;
# if 5432 is another local Postgres, use: POSTGRES_PORT=5433 bash scripts/apply-migrations.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS="${ROOT}/docker/postgres/migrations"

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-app}"

export PGPASSWORD="${POSTGRES_PASSWORD}"

psql_base=(psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=1)

if ! "${psql_base[@]}" -c "SELECT 1" >/dev/null 2>&1; then
	echo "psql: cannot connect to ${POSTGRES_HOST}:${POSTGRES_PORT} as ${POSTGRES_USER} (db=${POSTGRES_DB})." >&2
	echo "If you use Docker Compose on this machine, try: POSTGRES_PORT=5433 bash $0" >&2
	exit 1
fi

"${psql_base[@]}" -q -c "
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename text PRIMARY KEY,
	applied_at timestamptz NOT NULL DEFAULT now()
);"

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
	base=$(basename "$f")
	if [[ ! "$base" =~ ^[0-9]+_[a-zA-Z0-9_.-]+\.sql$ ]]; then
		echo "Refusing unsafe migration basename: ${base}" >&2
		exit 1
	fi
	applied=$("${psql_base[@]}" -tAc "SELECT COUNT(*) FROM schema_migrations WHERE filename = '$base';" | tr -d '[:space:]')
	if [[ "${applied}" == "1" ]]; then
		echo "Skipping (already applied): ${base}"
		continue
	fi
	echo "Applying ${base}"
	"${psql_base[@]}" -f "$f"
	"${psql_base[@]}" -c "INSERT INTO schema_migrations (filename) VALUES ('$base');"
done

echo "Migrations applied successfully."

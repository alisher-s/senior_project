#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

DATABASE_URL="${DATABASE_URL:-}"
if [[ -z "${DATABASE_URL}" ]]; then
  echo "DATABASE_URL is required (e.g. postgres://postgres:postgres@localhost:5432/app?sslmode=disable)" >&2
  exit 1
fi

MIGRATIONS_DIR="${ROOT}/docker/postgres/migrations"

migrate_cmd=(
  go run -tags "postgres"
  github.com/golang-migrate/migrate/v4/cmd/migrate
  -path "${MIGRATIONS_DIR}"
  -database "${DATABASE_URL}"
)

cmd="${1:-up}"
case "${cmd}" in
  up)
    "${migrate_cmd[@]}" up
    ;;
  down)
    n="${2:-}"
    if [[ -z "${n}" ]] || ! [[ "${n}" =~ ^[0-9]+$ ]]; then
      echo "usage: $0 down N" >&2
      exit 2
    fi
    "${migrate_cmd[@]}" down "${n}"
    ;;
  version)
    "${migrate_cmd[@]}" version
    ;;
  check)
    # Fails non-zero if the DB is in a dirty state.
    "${migrate_cmd[@]}" version >/dev/null
    # Must be idempotent: no pending migrations after applying.
    "${migrate_cmd[@]}" up
    ;;
  *)
    echo "usage: $0 [up|down N|version|check]" >&2
    exit 2
    ;;
esac


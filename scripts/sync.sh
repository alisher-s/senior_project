#!/usr/bin/env bash
# One-command teammate setup:
# - pull latest code
# - ensure Postgres container is running
# - apply SQL migrations (idempotent via schema_migrations table)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

echo "==> Pulling latest changes"
git pull --rebase

echo "==> Starting Postgres (Docker Compose)"
docker compose up -d postgres

echo "==> Applying DB migrations"
# Most dev setups map Postgres to 5433 (see scripts/apply-migrations.sh header).
POSTGRES_PORT="${POSTGRES_PORT:-5433}" bash scripts/apply-migrations.sh

echo "==> Done"


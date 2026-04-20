# Agent Instructions (senior_project)

## Goal
After each code change to the backend, refresh the running backend (using Docker via OrbStack) and run a small smoke-test suite to ensure endpoints still work.

## Database migrations (single happy-path schema)
- **Source of truth:** `docker/postgres/migrations/*.sql` only. Files run in **lexicographic order** (`001_…`, `002_…`, …). Do not duplicate schema elsewhere.
- **Docker:** `docker-compose` mounts that folder to `docker-entrypoint-initdb.d`. Scripts run **once**, when the Postgres data volume is first created. If you change SQL after a DB already exists, recreate the volume (`docker compose down -v`) or apply manually.
- **CI / existing DB:** `bash scripts/apply-migrations.sh` (needs `psql`). The script records applied files in table `schema_migrations` and skips them on re-run. Migrations are written to be idempotent where possible (`CREATE IF NOT EXISTS`, `ADD COLUMN IF NOT EXISTS`, etc.).
- **Host port:** Compose maps Postgres to **5432** and **5433** on the host. If `psql` fails with “role postgres does not exist” on the default port, another Postgres is bound to 5432—use `POSTGRES_HOST=127.0.0.1 POSTGRES_PORT=5433` (see `docker-compose.yml`).
- **Repositories** (`*/repository`) must match these tables/columns; change migrations and code together in one change set when possible.
- **Dev staff users (check-in / admin):** migration `006_dev_staff_users.sql` inserts `staff.admin@nu.edu.kz` (role `admin`) and `staff.organizer@nu.edu.kz` (role `organizer`). Shared password: `DevStaffPass1!`. For other accounts, an admin may call `PATCH /api/v1/admin/users/{id}/role` with body `{"role":"organizer"}` (or `admin` / `student`); all refresh tokens for that user are revoked so they must log in again.

## When to refresh
Refresh backend when changes affect any of these areas:
- `cmd/api/**`
- `internal/**`
- `auth/**`, `events/**`, `ticketing/**`, `payments/**`, `notifications/**`, `admin/**`, `analytics/**`

Do *not* refresh for docs-only changes (e.g. `README.md`) unless asked explicitly.

## Refresh backend (OrbStack + Docker Compose)
1. Ensure Docker is reachable:
   - Run `docker info` and ensure it returns successfully.
   - If it fails, instruct the user to start OrbStack (or their Docker service) and stop.
   - On macOS, try starting OrbStack automatically: `open -a "OrbStack"`, wait a few seconds, then re-run `docker info`.
2. Start/refresh dependencies:
   - `docker compose up -d postgres redis`
3. Rebuild + restart API:
   - `docker compose up --build -d api`
4. Wait until the API is reachable:
   - Poll `GET http://localhost:8080/api/v1/healthz` until it returns `200` (or fail with logs).

## Smoke tests (minimal but meaningful)
Use these checks to validate the system end-to-end:
1. Health
   - `GET /api/v1/healthz` must return `{"status":"ok"}` (or HTTP 200).
2. Auth flow
   - `POST /api/v1/auth/register`
   - `POST /api/v1/auth/login`
   - `POST /api/v1/auth/refresh` using the refresh token
3. Events + Ticketing flow (role-based)
   - `POST /api/v1/events` (create event)
   - `POST /api/v1/auth/login` as `staff.organizer@nu.edu.kz` / `DevStaffPass1!` when testing check-in (`POST /api/v1/tickets/use` requires organizer or admin).
   - `POST /api/v1/tickets/register` with `Authorization: Bearer <student_access_token>`
   - Repeat ticket registration for the same event+user and expect a non-201 status (prefer `409`).

## Compile check
Before smoke tests, run:
- `go test ./...`

If compilation fails, do not proceed to smoke tests.

## Update Swagger docs (required when API changes)
- Regenerate OpenAPI/Swagger spec (updates `docs/`):
  - `$(go env GOPATH)/bin/swag init -g cmd/api/main.go -o docs`

## Output / failure handling
- If health or any endpoint fails, include:
  - HTTP status code and response body (for the failing request)
  - the last relevant container logs via `docker compose logs --tail=200 api` (and postgres/redis if relevant)


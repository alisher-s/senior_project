# Agent Instructions (senior_project)

## Goal
After each code change to the backend, refresh the running backend (using Docker via OrbStack) and run a small smoke-test suite to ensure endpoints still work.

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
   - `POST /api/v1/tickets/register` with `Authorization: Bearer <access_token>`
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


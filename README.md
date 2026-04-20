# Student Event Ticketing Platform

HTTP API backend for university event discovery, moderation, ticketing with QR codes, and related admin/analytics flows—aimed at students, organizers, and administrators.

[![CI](https://github.com/alisher-s/senior_project/actions/workflows/ci.yml/badge.svg)](https://github.com/alisher-s/senior_project/actions/workflows/ci.yml)
![Go version](https://img.shields.io/badge/Go-1.25.3-00ADD8?logo=go)
![API version](https://img.shields.io/badge/OpenAPI-0.1-informational)

## Table of contents

1. [Project name & one-line description](#student-event-ticketing-platform)
2. [Badges](#student-event-ticketing-platform)
3. [Table of contents](#table-of-contents)
4. [Overview](#overview)
5. [Features](#features)
6. [Tech stack](#tech-stack)
7. [Project structure](#project-structure)
8. [Prerequisites](#prerequisites)
9. [Installation](#installation)
10. [Usage / quickstart](#usage--quickstart)
11. [Configuration](#configuration)
12. [API reference](#api-reference)
13. [Testing](#testing)
14. [Contributing](#contributing)
15. [License](#license)

## Overview

The service is a modular monolith: domain packages (`auth`, `events`, `ticketing`, etc.) register routes on a shared Chi router under `/api/v1`. It persists data in PostgreSQL, uses Redis for rate limiting (and analytics helpers), and optionally connects to MinIO for public event cover images. The problem it targets is operational event ticketing on campus: controlled registration, capacity, moderation before events go public, and check-in via QR hashes.

## Features

- **Authentication:** Register and login with email/password, JWT access tokens and refresh-token rotation (stored server-side), role claims (`student`, `organizer`, `admin`).
- **Email domain policy:** Registration restricted to a configurable NU email domain (default `nu.edu.kz`).
- **Organizer onboarding:** Students may request `organizer` via `PATCH /auth/me/roles`; admins change roles with audit-friendly refresh revocation.
- **Events:** CRUD for organizers/admins; public list/detail only shows **moderation-approved** events; search, date filters, pagination; optional `end_at`, location, capacity; cover image upload to MinIO when configured.
- **Admin moderation:** Approve/reject events, list pending events and users, moderation audit log, patch user roles.
- **Ticketing:** Student registration with concurrency-safe capacity, per user/event uniqueness, QR PNG (base64) + hash; cancel; organizer/admin check-in by QR hash with time-window and expiry rules; **expired** status computed in list responses when the event end instant has passed.
- **Notifications:** DB-backed email queue with background worker; SMTP sender when configured; ticket confirmation emails enqueued best-effort on registration; unauthenticated stub endpoint to enqueue mail.
- **Analytics:** Event registration/capacity stats for organizers (own events) and admins (any event or aggregate).
- **Payments:** `POST /payments/initiate` returns **501** (stub); webhook route validates HMAC-SHA256 using `PAYMENTS_WEBHOOK_SECRET`.
- **Ops:** Docker Compose for API, Postgres, Redis, MinIO; health check; Swagger UI; optional static file seeding to MinIO; Redis-backed rate limiting.

## Tech stack

| Area | Choice |
|------|--------|
| Language | Go 1.25.x (`go.mod`) |
| HTTP | `net/http`, [chi](https://github.com/go-chi/chi) v5 |
| Database | PostgreSQL 16 (driver: [pgx](https://github.com/jackc/pgx) v5) |
| Cache / rate limit | Redis 7 ([go-redis](https://github.com/redis/go-redis) v9) |
| Object storage | MinIO / S3-compatible ([minio-go](https://github.com/minio/minio-go) v7) |
| Auth | [golang-jwt](https://github.com/golang-jwt/jwt) v5, bcrypt ([x/crypto](https://golang.org/x/crypto)) |
| Validation | [go-playground/validator](https://github.com/go-playground/validator) v10 |
| QR codes | [skip2/go-qrcode](https://github.com/skip2/go-qrcode) |
| Email | [gopkg.in/gomail.v2](https://gopkg.in/gomail.v2) |
| API docs | [swaggo/swag](https://github.com/swaggo/swag), [http-swagger](https://github.com/swaggo/http-swagger) |
| Migrations (CLI) | [golang-migrate](https://github.com/golang-migrate/migrate) via `go run` in `scripts/migrate.sh` |
| Container images | `postgres:16-alpine`, `redis:7-alpine`, `minio/minio:latest` |

## Project structure

```text
.
├── admin/                 # Admin handlers, service, repository (moderation, user listing)
├── analytics/             # Event stats handlers, service, repository
├── auth/                  # Registration, login, refresh, roles; user repository
├── cmd/api/               # `main.go`: config, DB/Redis/MinIO, signal handling
├── docker/
│   └── postgres/
│       └── migrations/    # SQL migrations (Compose initdb + scripts)
├── docs/                  # Generated Swagger (`docs.go`, `swagger.json`, …)
├── events/                # Event CRUD, models, Postgres repository
├── internal/
│   ├── app/               # Chi router: middleware, module route registration, `/healthz`
│   ├── config/            # `LoadFromEnv`, typed configuration
│   └── infra/             # db, redis, http (CORS, logging), JWT middleware, rate limit, storage (MinIO)
├── notifications/         # Queue repo, email worker, SMTP sender, HTTP stub
├── payments/              # Stub service/repository, initiate + webhook handlers
├── scripts/               # `migrate.sh`, `apply-migrations.sh`, `sync.sh`
├── static/                # Optional images; copied in Docker; seedable to MinIO
├── ticketing/             # Ticket lifecycle, QR generation, Postgres repository
├── Dockerfile             # Multi-stage build; copies binary and `static/`
├── docker-compose.yml     # api, postgres, redis, minio
├── Makefile               # `migrate-up` / `migrate-down` wrappers
├── go.mod / go.sum        # Module `github.com/nu/student-event-ticketing-platform`
└── README.md
```

## Prerequisites

- **Go:** version **1.25.3** or compatible (see `go.mod`).
- **Docker** and **Docker Compose** (recommended for Postgres, Redis, MinIO, and the API image).
- **PostgreSQL client (`psql`)** if you use `scripts/apply-migrations.sh` on the host.
- **Network:** API default port **8080**; Postgres mapped to **5432** and **5433** on the host; Redis **6379**; MinIO **9000** (S3) and **9001** (console).

Environment requirements enforced in code (non-development):

- Non-empty `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`, `PAYMENTS_WEBHOOK_SECRET` (see [Configuration](#configuration); development may use defaults for JWT/payments if `APP_ENV=development`).

## Installation

```bash
git clone https://github.com/alisher-s/senior_project.git
cd senior_project
cp .env.example .env   # optional; Compose substitutes SMTP/MinIO vars
docker compose up -d postgres redis minio
# Apply schema (pick one):
# - Fresh volume: migrations already ran via Postgres init on first `up`
# - Existing DB:  DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5433/app?sslmode=disable' bash scripts/migrate.sh up
docker compose up --build -d api
```

Host-run alternative (after Postgres/Redis are up and schema applied):

```bash
export APP_ENV=development
export POSTGRES_HOST=127.0.0.1
export POSTGRES_PORT=5433    # use 5432 if that is your mapped port
export REDIS_HOST=127.0.0.1
export JWT_ACCESS_SECRET='...'
export JWT_REFRESH_SECRET='...'
export PAYMENTS_WEBHOOK_SECRET='...'
# MinIO: set MINIO_* or omit; cover upload falls back when storage is nil
go run ./cmd/api
```

## Usage / quickstart

1. Wait for `GET http://localhost:8080/api/v1/healthz` → `{"status":"ok"}`.
2. Open Swagger UI: `http://localhost:8080/api/v1/swagger/index.html`.
3. Register a student (email must match `AUTH_NU_EMAIL_DOMAIN`), create an event as organizer/admin, moderate as admin, register a ticket as student.

```bash
# Health
curl -sS http://localhost:8080/api/v1/healthz

# Register (use an @nu.edu.kz address by default)
curl -sS -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"student1@nu.edu.kz","password":"Password123!"}'

# Login — response includes access_token and refresh_token
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"student1@nu.edu.kz","password":"Password123!"}'
```

Pre-seeded staff (from migration `006_dev_staff_users.sql`), password **`DevStaffPass1!`**:

- `staff.admin@nu.edu.kz` — `admin`
- `staff.organizer@nu.edu.kz` — `organizer`

## Configuration

Variables are read from the process environment (e.g. Compose `environment` or a repo-root `.env` for substitution). Durations use Go’s `time.ParseDuration` (e.g. `15m`, `720h`).

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `APP_ENV` | string | `development` | Environment name; affects dev-only defaults for JWT and payments webhook secrets. |
| `PORT` | string | `8080` | HTTP listen port; server address is `:` + `PORT`. |
| `SERVER_READ_TIMEOUT` | duration | `10s` | `http.Server` read timeout (note: `main` also sets 5s read / 30s write on the server struct). |
| `SERVER_WRITE_TIMEOUT` | duration | `10s` | Intended write timeout in shared config. |
| `SERVER_IDLE_TIMEOUT` | duration | `60s` | Idle timeout. |
| `POSTGRES_HOST` | string | `postgres` | PostgreSQL host. |
| `POSTGRES_PORT` | int | `5432` | PostgreSQL port. |
| `POSTGRES_USER` | string | `postgres` | DB user. |
| `POSTGRES_PASSWORD` | string | `postgres` | DB password. |
| `POSTGRES_DB` | string | `app` | Database name. |
| `POSTGRES_SSLMODE` | string | `disable` | Passed to Postgres DSN (`sslmode`). |
| `REDIS_HOST` | string | `redis` | Redis host. |
| `REDIS_PORT` | int | `6379` | Redis port. |
| `REDIS_PASSWORD` | string | *(empty)* | Redis password, if any. |
| `REDIS_DB` | int | `0` | Redis logical DB index. |
| `JWT_ACCESS_SECRET` | string | *(required)* | HMAC secret for access tokens; in `development` only, empty falls back to `dev_access_secret_change_me`. |
| `JWT_REFRESH_SECRET` | string | *(required)* | HMAC secret for refresh tokens; dev fallback `dev_refresh_secret_change_me`. |
| `JWT_ACCESS_TTL` | duration | `15m` | Access token lifetime. |
| `JWT_REFRESH_TTL` | duration | `720h` (30d) | Refresh token lifetime. |
| `JWT_ISSUER` | string | `nu-ticketing` | JWT issuer claim. |
| `JWT_AUDIENCE` | string | `nu-ticketing-client` | JWT audience claim. |
| `RATE_LIMIT_REQUESTS` | int | `120` | Max requests per window (Redis sliding use). |
| `RATE_LIMIT_WINDOW_SECONDS` | int | `60` | Window length in seconds. |
| `AUTH_NU_EMAIL_DOMAIN` | string | `nu.edu.kz` | Allowed email domain for registration. |
| `AUTH_BCRYPT_COST` | int | `12` | bcrypt cost; must be ≥ 4. |
| `PAYMENTS_WEBHOOK_SECRET` | string | *(required)* | HMAC key for `POST /payments/webhook`; dev fallback `dev_payments_webhook_secret_change_me`. |
| `SMTP_HOST` | string | *(empty)* | SMTP host; empty disables real SMTP (worker uses no-op sender). |
| `SMTP_PORT` | int | `587` | SMTP port (`465` triggers implicit TLS in sender). |
| `SMTP_USER` | string | *(empty)* | SMTP auth user; if empty, `SMTP_FROM` is used. |
| `SMTP_FROM` | string | *(empty)* | From address; required with host/port/password for SMTP. |
| `SMTP_PASSWORD` | string | *(empty)* | SMTP password. |
| `MINIO_ENDPOINT` | string | *(required for storage)* | MinIO host:port reachable from the API process. |
| `MINIO_ACCESS_KEY` | string | *(required for storage)* | Access key. |
| `MINIO_SECRET_KEY` | string | *(required for storage)* | Secret key. |
| `MINIO_BUCKET` | string | *(required for storage)* | Bucket name (created if missing; public read policy set). |
| `MINIO_USE_SSL` | bool string | `false` | `true`/`false` for TLS to MinIO. |
| `MINIO_PUBLIC_URL` | string | *(required for storage)* | Public base URL for object URLs (no trailing slash trimmed internally). |
| `MINIO_SEED_STATIC` | string | *(enabled)* | Set `0`, `false`, or `no` to skip uploading `static/` on startup. |
| `MINIO_STATIC_SEED_DIR` | string | *(optional)* | Override directory to seed instead of `./static` or beside the binary. |
| `DATABASE_URL` | string | — | **Required** by `scripts/migrate.sh` / `make migrate-up`: full Postgres DSN for golang-migrate. |
| `DATABASE_URL_TEST` | string | — | Used in CI for tests that need a DSN string (see workflow). |

## API reference

Base path: **`/api/v1`**. Errors: JSON `{"error":{"code":"string","message":"string"}}`.

| Method | Path | Auth / roles | Description |
|--------|------|--------------|-------------|
| GET | `/healthz` | — | Liveness; returns `{"status":"ok"}`. |
| GET | `/swagger/*` | — | Swagger UI + `doc.json`. |
| GET | `/static/*` | — | Served only if a `static` directory exists at process CWD (StripPrefix from `/api/v1/static/`). |
| POST | `/auth/register` | — | Create user; returns tokens and user. |
| POST | `/auth/login` | — | Returns tokens and user. |
| POST | `/auth/refresh` | — | Rotate refresh token; returns new tokens. |
| PATCH | `/auth/me/roles` | JWT | Student requests `{"roles":["organizer"]}`. |
| POST | `/events` | JWT organizer/admin | Create event (starts `pending` moderation). |
| GET | `/events` | — | List **approved** events; query: `q`, `limit`, `offset`, `starts_after`, `starts_before` (RFC3339). |
| GET | `/events/mine` | JWT organizer/admin | Dashboard list; admin optional `organizer_id`. |
| GET | `/events/{id}` | — | Get approved event by ID. |
| POST | `/events/{id}/cover-image` | JWT organizer/admin | Multipart cover upload to MinIO. |
| PUT | `/events/{id}` | JWT organizer/admin | Update event fields. |
| DELETE | `/events/{id}` | JWT organizer/admin | Delete event. |
| GET | `/tickets/my` | JWT | Current user’s tickets (`status` may include computed `expired`). |
| POST | `/tickets/register` | JWT student | Register for event; returns QR base64 + hash. |
| POST | `/tickets/{id}/cancel` | JWT student | Cancel ticket. |
| POST | `/tickets/use` | JWT organizer/admin | Check-in body: `qr_hash_hex`. |
| POST | `/payments/initiate` | JWT student/organizer/admin | **501** stub (`not_implemented`). |
| POST | `/payments/webhook` | — | Webhook; HMAC SHA-256 hex in `X-Signature` header. |
| POST | `/notifications/send-email` | — | Enqueue email (`to`, `title`, `body`); may return 501 if not wired. |
| GET | `/admin/events` | JWT admin | List events; `moderation_status`, `q`, `limit`, `offset`. |
| GET | `/admin/users` | JWT admin | List users; `q` (email substring), `limit`, `offset`. |
| POST | `/admin/events/{id}/moderate` | JWT admin | Approve/reject with reason. |
| PATCH | `/admin/users/{id}/role` | JWT admin | Set role (`student` / `organizer` / `admin`). |
| GET | `/admin/moderation-logs` | JWT admin | Paginated audit log; filters `event_id`, `admin_id`. |
| GET | `/analytics/events/stats` | JWT organizer/admin | Stats; optional `event_id`. |

Request/response schemas and tags are defined in the Swagger annotations and generated files under `docs/`.

## Testing

```bash
go test ./... -race -count=1
```

CI (`.github/workflows/ci.yml`) runs **golangci-lint** and **`go test ./... -race -count=1`** against a Postgres and Redis service, after `bash scripts/migrate.sh up` and `check`.

**Packages with tests (non-exhaustive):**

- `internal/infra/auth` — JWT unit tests.
- `auth/service` — organizer request flow.
- `internal/notifications/sender` — SMTP helper tests.
- `ticketing/service` — email HTML helper; foundation integration tests (skip if DB unavailable).
- `internal/app/ticketing` — **integration** tests hitting a real router/DB/Redis; default `POSTGRES_PORT=5433` and `127.0.0.1` unless overridden.

For integration tests locally, start dependencies (`docker compose up -d postgres redis`), ensure schema is applied, then run `go test` with the same env vars as CI or the defaults in the test file.

## Contributing

- **Branches:** CI runs on **`main`** and **`master`** for pushes and pull requests.
- **Before opening a PR:** run `go test ./... -race -count=1` and fix **golangci-lint** findings (matches CI).
- **API changes:** regenerate Swagger from the repo root:  
  `$(go env GOPATH)/bin/swag init -g cmd/api/main.go -o docs`
- **Database:** keep `docker/postgres/migrations` as the source of truth; update repositories to match. For existing volumes, recreate (`docker compose down -v`) or apply migrations manually / via `scripts/migrate.sh` or `scripts/apply-migrations.sh` as appropriate.
- **Style:** follow existing patterns in domain packages (Chi handlers, `httpx` helpers, `authx` middleware).

## License

No `LICENSE` file is present in this repository; all rights are reserved unless the maintainers add a license.

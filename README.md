# Student Event Ticketing Platform — Backend API

## Quick start (local development)

1. **Clone the repository**

```bash
git clone <your-repo-url> senior_project
cd senior_project
```

2. **Environment file**

```bash
cp .env.example .env
```

Edit `.env` if you need SMTP for outbound email (optional in local dev). Docker Compose also reads variables from this file for the `api` service.

3. **Start the stack**

```bash
docker compose up --build
```

This builds the API image, starts **api** (port **8080**), **postgres**, and **redis**.

4. **Verify the API**

```bash
curl -sS http://localhost:8080/api/v1/healthz
```

Expected JSON: `{"status":"ok"}`.

5. **Check Postgres and Redis**

```bash
docker compose ps
```

You should see `postgres`, `redis`, and `api` running. Ports on the host:

- **PostgreSQL:** `localhost:5432` (and `localhost:5433` maps to the same container for tools that need an alternate host port)
- **Redis:** `localhost:6379`

Optional connectivity checks:

```bash
nc -zv localhost 5432
nc -zv localhost 6379
```

6. **Swagger UI**

Open: **http://localhost:8080/api/v1/swagger/index.html**

---

## Environment variables

Variables are loaded from the process environment (e.g. `.env` with Docker Compose, or your shell for `go run`). Defaults below match `internal/config/config.go`.

### Server

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `APP_ENV` | No | `development` | Environment name. In `development`, empty JWT/payment webhook secrets get safe dev defaults. |
| `PORT` | No | `8080` | HTTP listen port (server binds to `:{PORT}`). |
| `SERVER_READ_TIMEOUT` | No | `10s` | Server read timeout (Go duration string). |
| `SERVER_WRITE_TIMEOUT` | No | `10s` | Server write timeout. |
| `SERVER_IDLE_TIMEOUT` | No | `60s` | Server idle timeout. |

### Database

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_HOST` | No | `postgres` | Postgres hostname (use `localhost` when running the API on the host). |
| `POSTGRES_PORT` | No | `5432` | Postgres port. |
| `POSTGRES_USER` | No | `postgres` | Database user. |
| `POSTGRES_PASSWORD` | No | `postgres` | Database password. |
| `POSTGRES_DB` | No | `app` | Database name. |
| `POSTGRES_SSLMODE` | No | `disable` | Passed to the Postgres driver (use `disable` locally). |

### Redis

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `REDIS_HOST` | No | `redis` | Redis hostname (use `localhost` when running the API on the host). |
| `REDIS_PORT` | No | `6379` | Redis port. |
| `REDIS_PASSWORD` | No | *(empty)* | Redis password, if configured. |
| `REDIS_DB` | No | `0` | Redis logical DB index. |

### Auth

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_ACCESS_SECRET` | Yes (unless `APP_ENV=development` with empty value) | *(empty; dev default applied)* | HMAC secret for access tokens. |
| `JWT_REFRESH_SECRET` | Yes (unless `APP_ENV=development` with empty value) | *(empty; dev default applied)* | HMAC secret for refresh tokens. |
| `JWT_ACCESS_TTL` | No | `15m` | Access token lifetime (Go duration, e.g. `15m`). |
| `JWT_REFRESH_TTL` | No | `720h` | Refresh token lifetime (default 30 days). |
| `JWT_ISSUER` | No | `nu-ticketing` | JWT `iss` claim. |
| `JWT_AUDIENCE` | No | `nu-ticketing-client` | JWT `aud` claim. |
| `AUTH_NU_EMAIL_DOMAIN` | No | `nu.edu.kz` | Allowed email domain for `POST /auth/register`. |
| `AUTH_BCRYPT_COST` | No | `12` | Bcrypt cost for password hashing (minimum 4). |
| `RATE_LIMIT_REQUESTS` | No | `120` | Max requests per client per route window before **429**. |
| `RATE_LIMIT_WINDOW_SECONDS` | No | `60` | Sliding window length in seconds for rate limiting. |

### SMTP

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SMTP_HOST` | No | *(empty)* | SMTP server host; if empty, the email worker uses a no-op sender. |
| `SMTP_PORT` | No | `587` | SMTP port. |
| `SMTP_USER` | No | *(empty)* | SMTP username for auth. If empty, the app uses `SMTP_FROM` as the username. |
| `SMTP_FROM` | No | *(empty)* | From address for outbound mail. |
| `SMTP_PASSWORD` | No | *(empty)* | SMTP password. |

### Payments

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PAYMENTS_WEBHOOK_SECRET` | Yes (unless `APP_ENV=development` with empty value) | *(empty; dev default applied)* | Shared secret for **HMAC-SHA256** verification of `POST /payments/webhook` bodies (`X-Signature`). |

---

## Authentication

The API uses **JWT access tokens** in the `Authorization` header and **refresh tokens** stored server-side (identified by `jti` inside the JWT). Access and refresh tokens use different signing secrets.

### 1. Register — `POST /api/v1/auth/register`

**Request body (JSON):**

| Field | Type | Required | Rules |
|-------|------|----------|--------|
| `email` | string | Yes | Valid email ending with `@` + `AUTH_NU_EMAIL_DOMAIN` (default `nu.edu.kz`). |
| `password` | string | Yes | Length 8–72. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"student@nu.edu.kz","password":"verystrongpassword"}'
```

**Success: HTTP 201** — body shape:

```json
{
  "access_token": "<jwt>",
  "refresh_token": "<jwt>",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "student@nu.edu.kz",
    "role": "student",
    "roles": ["student"]
  }
}
```

(`pending_roles` may appear when organizer approval is pending.)

**Errors:** **400** `invalid_request`; **409** `email_exists`; **400** `email_not_allowed`.

### 2. Login — `POST /api/v1/auth/login`

Same request fields as register.

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"student@nu.edu.kz","password":"verystrongpassword"}'
```

**Success: HTTP 200** — same `AuthResponseDTO` shape as register (`access_token`, `refresh_token`, `user`).

**Errors:** **401** `invalid_credentials`.

### 3. Using the access token

Send on every protected request:

```http
Authorization: Bearer <access_token>
```

Example:

```bash
curl -sS http://localhost:8080/api/v1/tickets/my \
  -H "Authorization: Bearer <access_token>"
```

### 4. Refresh — `POST /api/v1/auth/refresh`

**Request body:**

| Field | Type | Required |
|-------|------|----------|
| `refresh_token` | string | Yes |

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token_from_login_or_register>"}'
```

**Success: HTTP 200** — new `access_token` and **new** `refresh_token` (`user` included).

**Refresh token semantics:** each refresh token can be used **once**. The server **consumes** the old refresh token (`jti`) and issues a new refresh token. Reusing the same refresh token returns **401** with `refresh_token_consumed`.

### 5. Token expiry

| Token | TTL (default) | Source |
|-------|----------------|--------|
| Access | **15 minutes** | `JWT_ACCESS_TTL` (default `15m` in `internal/config/config.go`) |
| Refresh | **30 days** | `JWT_REFRESH_TTL` (default `720h`) |

### Roles

| Role | Meaning |
|------|---------|
| `student` | Default role from registration; can browse approved events, register/cancel own tickets. |
| `organizer` | Can create/update/delete own events (subject to rules), scan QR at check-in, view analytics for own events. |
| `admin` | Full moderation and user role management; can moderate any event; analytics for all events. |

**What each role can do (simplified):**

| Capability | student | organizer | admin |
|------------|---------|-----------|-------|
| Register / login / refresh | Yes | Yes | Yes |
| `GET /events`, `GET /events/{id}` (approved only) | Yes | Yes | Yes |
| `POST /events`, `PUT/DELETE /events/{id}` | No | Own events only | Any event |
| `POST /tickets/register`, cancel own ticket | Yes | No* | No* |
| `POST /tickets/use` (QR check-in) | No | Yes | Yes |
| `POST /admin/...`, `GET /admin/moderation-logs` | No | No | Yes |
| `PATCH /admin/users/{id}/role` | No | No | Yes |
| `GET /analytics/events/stats` | No | Own events | All events |
| Request organizer via `PATCH /auth/me/roles` | Yes (`{"roles":["organizer"]}`) | N/A | N/A |

\*Organizer/admin accounts are not intended to use student-only ticket routes; middleware requires role `student` for registration.

---

## API reference

Unless noted, send `Content-Type: application/json`. Base URL: `http://localhost:{PORT}/api/v1` (default port **8080**).

### Auth endpoints

#### `POST /api/v1/auth/register`

| | |
|--|--|
| **Auth** | No |
| **Request body** | See [Register](#1-register--post-apiv1authregister) — `email` (string, required), `password` (string, required, min 8). |
| **Success** | **201** — `access_token`, `refresh_token`, `user` (see Authentication). |
| **Errors** | **400** invalid JSON/validation (`invalid_request`); **400** `email_not_allowed`; **409** `email_exists`; **500** `internal_error`. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"newuser@nu.edu.kz","password":"verystrongpassword"}'
```

#### `POST /api/v1/auth/login`

| | |
|--|--|
| **Auth** | No |
| **Request body** | `email` (string, required), `password` (string, required, min 8). |
| **Success** | **200** — same as register. |
| **Errors** | **400** `invalid_request`; **401** `invalid_credentials`; **500** `internal_error`. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"student@nu.edu.kz","password":"verystrongpassword"}'
```

#### `POST /api/v1/auth/refresh`

| | |
|--|--|
| **Auth** | No |
| **Request body** | `refresh_token` (string, required). |
| **Success** | **200** — new `access_token`, `refresh_token`, `user`. |
| **Errors** | **400** `invalid_request`; **401** `invalid_refresh_token` or `refresh_token_consumed`; **500** `internal_error`. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<paste_refresh_token>"}'
```

#### `PATCH /api/v1/auth/me/roles` (organizer request)

| | |
|--|--|
| **Auth** | Yes — Bearer access token |
| **Roles** | Active **student** (must send exactly `{"roles":["organizer"]}`) |
| **Request body** | `roles` (array of strings, required) — must be exactly `["organizer"]`. |
| **Success** | **200** — `{ "user": { ... } }` |
| **Errors** | **400** `invalid_request`; **403** `organizer_request_forbidden`; **409** `organizer_already_active`; **401** JWT errors. |

---

### Event endpoints

#### `GET /api/v1/events`

| | |
|--|--|
| **Auth** | No |
| **Query parameters** | See below |
| **Success** | **200** — `{ "items": [ EventDTO ... ], "limit": int, "offset": int }` |

**Query parameters:**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `q` | string | No | Case-insensitive substring match on **`title`** only (`ILIKE`). |
| `limit` | integer | No | Page size; **default 20** if omitted or invalid low; must be **1–100** if provided. |
| `offset` | integer | No | Offset; default **0**; must be **0–100000** if provided. |
| `starts_after` | string (RFC3339) | No | Only events with `starts_at` **after** this instant. |
| `starts_before` | string (RFC3339) | No | Only events with `starts_at` **before or equal** to this instant. |

Invalid `limit`/`offset` or invalid RFC3339 dates → **400** `invalid_request`.

**Note:** Only events with **`moderation_status=approved`** are returned.

```bash
curl -sS "http://localhost:8080/api/v1/events?limit=10&offset=0&q=hackathon&starts_after=2026-01-01T00:00:00Z"
```

#### `POST /api/v1/events`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `organizer`, `admin` |
| **Request body** | `title` (string, required, 3–120), `description` (string, max 2000), `cover_image_url` (string, optional, max 2048), `starts_at` (string/time, RFC3339, required), `capacity_total` (integer, required, 1–100000). |
| **Success** | **201** — `EventDTO` |
| **Errors** | **401** / **403**; **400** `invalid_request`; **500** `internal_error`. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <organizer_access_token>" \
  -d '{"title":"NU Hackathon","description":"Annual hackathon","starts_at":"2026-06-01T10:00:00Z","capacity_total":100,"cover_image_url":"https://example.com/cover.jpg"}'
```

#### `GET /api/v1/events/{id}`

| | |
|--|--|
| **Auth** | No |
| **Success** | **200** — `EventDTO` |
| **Errors** | **400** `invalid_id`; **404** `not_found` if missing or **not approved** for public view. |

```bash
curl -sS "http://localhost:8080/api/v1/events/550e8400-e29b-41d4-a716-446655440000"
```

#### `PUT /api/v1/events/{id}`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `organizer` (own event only), `admin` (any) |
| **Request body** | All optional: `title`, `description`, `cover_image_url` (use `""` to clear cover), `starts_at`, `capacity_total`, `status` (`draft` \| `published` \| `cancelled`). |
| **Success** | **200** — `EventDTO` |
| **Errors** | **403** `forbidden`; **404** `not_found`; **400** `invalid_request`. |

```bash
curl -sS -X PUT http://localhost:8080/api/v1/events/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <organizer_access_token>" \
  -d '{"title":"NU Hackathon 2026","status":"published"}'
```

#### `DELETE /api/v1/events/{id}`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `organizer` (own), `admin` (any) |
| **Success** | **204** No Content |
| **Errors** | **403**; **404**; **500**. |

```bash
curl -sS -X DELETE http://localhost:8080/api/v1/events/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <organizer_access_token>"
```

---

### Ticketing endpoints

#### `GET /api/v1/tickets/my`

| | |
|--|--|
| **Auth** | Yes (any authenticated user) |
| **Success** | **200** — `{ "tickets": [ { "ticket_id", "status", "qr_hash_hex", "event_id", "event_title", "event_date" } ] }` |
| **Errors** | **401**. |

```bash
curl -sS http://localhost:8080/api/v1/tickets/my \
  -H "Authorization: Bearer <access_token>"
```

#### `POST /api/v1/tickets/register`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `student` |
| **Request body** | `event_id` (string UUID, required). |
| **Success** | **201** — `ticket_id`, `event_id`, `user_id`, `status`, `qr_png_base64`, `qr_hash_hex` |
| **Errors** | **400**; **404** `not_found`; **409** `capacity_full`, `already_registered`, `event_not_approved`, `event_not_published`, `event_cancelled`, `registration_closed`, etc. |

**QR flow:**

- **`qr_hash_hex`:** SHA-256 of the ticket’s random **payload**, written as lowercase **hex** (64 characters). The database stores **only this hash**, not the payload.
- **`qr_png_base64`:** Standard Base64-encoded PNG (no `data:image/png;base64,` prefix). The QR image encodes the **payload string** (not the hash).

**Organizer scan:** read the **payload** from the QR (the same string that was hashed at issuance), compute **SHA-256 → hex**, and send that value as `qr_hash_hex` in `POST /tickets/use`. You can also send the `qr_hash_hex` returned by `POST /tickets/register` if the attendee app displays it or your client stored it (it must match the DB row).

Then call:

```bash
curl -sS -X POST http://localhost:8080/api/v1/tickets/use \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <organizer_access_token>" \
  -d '{"qr_hash_hex":"<64-char-hex-or-value-from-register>"}'
```

#### `POST /api/v1/tickets/{id}/cancel`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `student` |
| **Path** | `id` — ticket UUID |
| **Success** | **200** — `ticket_id`, `event_id`, `user_id`, `status` |
| **Errors** | **409** `ticket_already_cancelled`, `cancellation_not_allowed`; **404**; **401**. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/tickets/550e8400-e29b-41d4-a716-446655440000/cancel \
  -H "Authorization: Bearer <student_access_token>"
```

#### `POST /api/v1/tickets/use`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `organizer`, `admin` |
| **Request body** | `qr_hash_hex` (string, required). |
| **Success** | **200** — `ticket_id`, `event_id`, `user_id`, `status` (`used`) |
| **Errors** | **404** `ticket_not_found`; **400** `ticket_expired` (event end instant has passed; **expires strictly after** `end_at` — `end_at` is **inclusive**); **409** `ticket_already_used`, `check_in_not_open`, `ticket_cannot_be_used`, etc. |

---

### Payment endpoints

> **Important:** The payment **repository is currently a stub**. `POST /payments/initiate` and `POST /payments/webhook` return **501 Not Implemented** with `error.code` **`not_implemented`** until a real payment backend is wired. Do not treat these as production-ready.

#### `POST /api/v1/payments/initiate`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `student`, `organizer`, `admin` |
| **Request body** | `event_id` (string, required), `amount` (integer, required, `> 0`), `currency` (string, required, exactly 3 letters). |
| **Success** | **201** — `payment_id`, `provider_ref`, `provider_url` *(when implemented)* |
| **Errors** | **501** `not_implemented` *(current stub)*; **400**; **401**; **500**. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/payments/initiate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{"event_id":"550e8400-e29b-41d4-a716-446655440000","amount":1000,"currency":"KZT"}'
```

**Intended flow (when payments are enabled):** for paid events, clients would register for an event, call initiate, open `payment_url`, and the provider would call the webhook; on success, ticketing would issue the ticket and QR. **Today:** ticket registration does not depend on this stub; **`POST /tickets/register`** issues a ticket with status **`active`** and QR when business rules pass. There is **no** separate `pending_payment` ticket status in the API schema.

#### `POST /api/v1/payments/webhook`

| | |
|--|--|
| **Auth** | No (provider callback; not for browsers) |
| **Headers** | **`X-Signature`**: hex-encoded **HMAC-SHA256** of the **raw** request body using `PAYMENTS_WEBHOOK_SECRET`. |
| **Request body** | `provider_ref` (string, required), `status` (string, required). |
| **Success** | **200** — `{}` |
| **Errors** | **401** `missing_signature`; **403** `invalid_signature`; **404** `not_found`; **501** `not_implemented` *(stub)*; **400** `invalid_request`. |

```bash
BODY='{"provider_ref":"ref-123","status":"succeeded"}'
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "dev_payments_webhook_secret_change_me" -binary | xxd -p -c 256)
curl -sS -X POST http://localhost:8080/api/v1/payments/webhook \
  -H "Content-Type: application/json" \
  -H "X-Signature: $SIG" \
  -d "$BODY"
```

---

### Notification endpoints

#### `POST /api/v1/notifications/send-email`

| | |
|--|--|
| **Auth** | No |
| **Request body** | `to` (email, required), `title` (string, 3–200 chars), `body` (string, 1–5000 chars). |
| **Success** | **202** Accepted (email enqueued; empty body) |
| **Errors** | **400** `invalid_request`; **500** `internal_error`. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/notifications/send-email \
  -H "Content-Type: application/json" \
  -d '{"to":"user@nu.edu.kz","title":"Hello","body":"Queued notification body."}'
```

---

### Admin endpoints

#### `POST /api/v1/admin/events/{id}/moderate`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `admin` |
| **Request body** | `action` (string, required): `approve` or `reject`; `reason` (string, optional, max 2000). |
| **Success** | **200** — `{ "moderation_status": "approved" | "rejected" }` |
| **Errors** | **400** `invalid_id`, `invalid_action`; **404** `not_found`; **401**; **403** `forbidden`. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/admin/events/550e8400-e29b-41d4-a716-446655440000/moderate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_access_token>" \
  -d '{"action":"approve","reason":"Looks good"}'
```

#### `PATCH /api/v1/admin/users/{id}/role`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `admin` |
| **Request body** | `role` (string): `student`, `organizer`, or `admin`. |
| **Success** | **200** — `id`, `email`, `role` |
| **Errors** | **400** `invalid_role`; **404** `not_found`; **401**; **403**. |

*Changing roles revokes existing refresh tokens in the backend; users must **log in again** for a new refresh token.*

```bash
curl -sS -X PATCH http://localhost:8080/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000/role \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_access_token>" \
  -d '{"role":"organizer"}'
```

#### `GET /api/v1/admin/moderation-logs`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `admin` |
| **Query** | `event_id` (UUID, optional), `admin_id` (UUID, optional), `limit` (default 20, max 100), `offset` (default 0). |
| **Success** | **200** — `{ "items": [...], "limit": int, "offset": int }` |

```bash
curl -sS "http://localhost:8080/api/v1/admin/moderation-logs?limit=20&offset=0" \
  -H "Authorization: Bearer <admin_access_token>"
```

---

### Analytics endpoints

#### `GET /api/v1/analytics/events/stats`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `organizer`, `admin` |
| **Query** | `event_id` (UUID, optional) — omit to aggregate events in scope (organizer: own events; admin: all). |
| **Success** | **200** — `event_id` (optional string), `total_capacity`, `registered_count`, `remaining_capacity`, `registration_timeline` (array of `{ "hour", "count" }`), `as_of` (RFC3339). |
| **Errors** | **403** `forbidden` (organizer viewing another organizer’s event); **404** `not_found`; **401**. |

```bash
curl -sS "http://localhost:8080/api/v1/analytics/events/stats?event_id=550e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer <organizer_access_token>"
```

---

## Data models

JSON field names match API responses. **Nullable** fields are noted.

### User (in auth responses)

```json
{
  "id": "uuid-string",
  "email": "string",
  "role": "student | organizer | admin",
  "roles": ["string"],
  "pending_roles": ["organizer"]
}
```

- `pending_roles`: optional; present when a role is awaiting approval (e.g. organizer request).

### Event (`EventDTO`)

```json
{
  "id": "string (UUID)",
  "title": "string",
  "description": "string",
  "cover_image_url": "string",
  "starts_at": "RFC3339 datetime",
  "capacity_total": 0,
  "capacity_available": 0,
  "status": "draft | published | cancelled",
  "moderation_status": "pending | approved | rejected"
}
```

- `cover_image_url`: omitted or empty when not set (`omitempty`).

### Ticket (register / cancel / use responses; list item shapes differ slightly)

```json
{
  "ticket_id": "string (UUID)",
  "event_id": "string (UUID)",
  "user_id": "string (UUID)",
  "status": "active | used | cancelled | expired",
  "qr_png_base64": "string (standard Base64 PNG, register only)",
  "qr_hash_hex": "string (hex)"
}
```

**`GET /tickets/my` item:** `ticket_id`, `status`, `qr_hash_hex`, `event_id`, `event_title`, `event_date` (RFC3339 string). No `qr_png_base64` in list. The value **`expired`** is returned when the ticket is still **`active`** in the database but the event end instant (`end_at` if set, otherwise `starts_at`) is in the past (not stored as a DB status).

### Payment (when implemented; stub returns 501 today)

```json
{
  "payment_id": "string (UUID)",
  "provider_ref": "string",
  "provider_url": "string",
  "amount": 0,
  "currency": "string",
  "status": "pending | succeeded | failed | canceled"
}
```

Initiate response currently only documents `payment_id`, `provider_ref`, `provider_url` in the handler DTO.

### Notification (queue / outbound email)

Internal queue row (for context; HTTP enqueue does not return the full row):

```json
{
  "id": "string",
  "type": "email | push",
  "to": "string (recipient)",
  "title": "string",
  "body": "string",
  "status": "queued | processing | sent | failed"
}
```

---

## Error handling

### Standard error JSON

```json
{
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

There is **no** `fields` array: validation failures use **`invalid_request`** with a **single human-readable `message`** (from JSON decode or go-playground validator).

### Common `error.code` values

| HTTP | Code | When |
|------|------|------|
| **400** | `invalid_request` | Invalid JSON, unknown fields (where disallowed), or failed struct validation (missing fields, wrong types, tag violations). Example: `{"error":{"code":"invalid_request","message":"Key: 'RegisterRequestDTO.Password' Error:Field validation for 'Password' failed on the 'min' tag"}}` |
| **400** | `invalid_id` | Malformed UUID in path or body. |
| **400** | `ticket_expired` | QR check-in **strictly after** the event end instant (`end_at` if set, otherwise `starts_at`; `end_at` is **inclusive**). |
| **400** | `email_not_allowed` | Registration email domain not allowed. |
| **400** | `invalid_role` / `invalid_action` | Admin or moderation validation. |
| **401** | `missing_authorization`, `invalid_authorization`, `invalid_token`, `invalid_token_claims` | Missing/invalid Bearer token or claims. |
| **401** | `invalid_credentials` | Wrong password on login. |
| **401** | `invalid_refresh_token`, `refresh_token_consumed` | Refresh misuse or reuse. |
| **401** | `missing_signature` | Webhook without `X-Signature`. |
| **403** | `forbidden` | Authenticated but role not allowed (RBAC). |
| **403** | `invalid_signature` | Webhook HMAC verification failed. |
| **403** | `organizer_request_forbidden` | Non-student requested organizer role. |
| **404** | `not_found` | Entity missing or hidden (e.g. unapproved event for public `GET`). |
| **409** | `email_exists`, `already_registered`, `capacity_full`, … | Business conflicts (see handlers). |
| **429** | `rate_limited` | Too many requests; check `Retry-After`. |
| **501** | `not_implemented` | Payment (and similar) not enabled. |

### Client handling

- Read `response.status` and parse JSON `error.code` / `error.message`.
- On **401** with expired access token, call **`POST /auth/refresh`** then retry once.
- On **429**, honor **`Retry-After`** (seconds) before retrying.
- On **501** for payments, hide pay flows or show “not available” — do not assume success.

---

## Event moderation flow

1. **Organizer or admin** creates an event (`POST /events`). New rows get **`moderation_status=pending`** (per database default) and are **not** visible on public `GET /events` or `GET /events/{id}` until approved.
2. **Admin** calls `POST /admin/events/{id}/moderate` with `approve` or `reject`.
3. **Public listings** only include **`moderation_status=approved`**. Pending or rejected events behave like “not found” for public GET by id.

Event **`status`** (`draft` / `published` / `cancelled`) is separate from moderation: ticketing also enforces published/not cancelled and approved moderation for registration.

**Frontend/mobile:** show only events from `GET /events` for browse screens; organizer dashboards should use authenticated flows or admin tools to see pending/rejected items (the public API does not expose non-approved events in list/detail).

---

## Ticket lifecycle

Current ticket **`status`** values in the API: **`active`**, **`used`**, **`cancelled`**, and **`expired`** (computed in responses when the event has ended; not stored in `tickets.status`).

```
                    +------------------+
                    | POST /tickets/   |
                    | register         |
                    +--------+---------+
                             |
                             v
                      +-------------+
                      |   active    |  (QR issued in response)
                      +------+------+
                             |
         +-------------------+-------------------+
         |                   |                   |
         v                   v                   v
  (event ended,        +-----------+    +------------------+
   list/read-only)     | POST .../ |    | POST /tickets/use|
         |             | cancel    |    | (organizer/admin)|
         v             +-----+-----+    +--------+---------+
    +----------+             |                 |
    | expired* |             v                 v
    +----------+      +-------------+   +-------------+
                        | cancelled   |   |    used     |
                        +-------------+   +-------------+
```

\* **`expired`** appears in **`GET /tickets/my`** (and similar) when the stored status is **`active`** but **`now` is strictly after** the event end instant (`events.end_at` if set, otherwise `events.starts_at`). The instant itself is **inclusive** (i.e., not expired when `now == end_instant`). Check-in is rejected with **`ticket_expired`**.

- **`active`:** valid ticket; show QR (`qr_png_base64` / `qr_hash_hex` from registration).
- **`expired`:** same row as active in DB; API surfaces **`expired`** after the event end time so clients can hide or disable QR.
- **`used`:** check-in completed; show as “used” / hide QR for re-entry per product rules.
- **`cancelled`:** show as cancelled.

---

## One-command teammate sync (pull + DB migrations)

From the repo root:

```bash
bash scripts/sync.sh
```

If your Postgres is exposed on a different host port (e.g. `5432`), run:

```bash
POSTGRES_PORT=5432 bash scripts/sync.sh
```

**Payments:** there is no `pending_payment` status on tickets in this API. **`POST /payments/initiate`** is currently **501**. Free vs paid pricing is not modeled on events; all successful registrations follow the flow above.

---

## Running without Docker

Requirements: **Go 1.22+**, **PostgreSQL**, **Redis** (same schema/migrations as Docker).

1. Run migrations before starting the server: `make migrate-up`
2. Export environment variables. **Minimum to start** (non-development) per `LoadFromEnv`:

- `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET` (non-empty)
- `PAYMENTS_WEBHOOK_SECRET` (non-empty)
- `POSTGRES_*` pointing at your DB
- Redis reachable via `REDIS_*`

With `APP_ENV=development`, empty JWT secrets and empty `PAYMENTS_WEBHOOK_SECRET` are replaced by dev defaults (not for production).

3. Run:

```bash
go run ./cmd/api
```

If Postgres is on **`localhost:5433`** (Docker mapped port), set e.g. `POSTGRES_HOST=localhost` and `POSTGRES_PORT=5433`.

---

## CORS

Allowed **browser** origins (see `internal/infra/http/middleware.go`):

- `http://localhost:3000`
- `http://localhost:5173`

Methods: `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`. Headers: `Content-Type`, `Authorization`. Credentials allowed when origin matches.

**Mobile (Flutter)** and other native clients typically do not send a browser `Origin`; CORS does not apply. Use your machine’s **LAN IP** and port (e.g. `http://192.168.1.10:8080`) instead of `localhost` when testing on a physical device.

---

## Development notes for frontend (React)

- **Base URL:** `http://localhost:{PORT}/api/v1` (default **8080**).
- **Tokens:** keep **`access_token` in memory** (not `localStorage` if you want to reduce XSS risk); store **`refresh_token`** in an **HttpOnly cookie** (if you add a BFF) or secure storage appropriate to your threat model.
- **401 handling:** on **401**, call **`POST /auth/refresh`**, update tokens, **retry the request once**; if refresh fails, redirect to login.
- **Pagination:** `GET /events` uses **`limit`** (default 20, max 100) and **`offset`**; response echoes `limit` and `offset` for UI state.
- **Images:** `cover_image_url` is a **plain HTTPS URL string** — upload files to your own storage/CDN, then pass the URL in `POST`/`PUT` events.

---

## Development notes for mobile (Flutter)

- Use **`dio`** or **`http`** with an **interceptor** that adds `Authorization: Bearer <access_token>` to API calls.
- On **401**, run **refresh** in the interceptor and **retry** once.
- **QR display:** `qr_png_base64` is raw Base64 PNG → `Image.memory(base64Decode(ticket.qr_png_base64))` (add `data:` prefix only if you choose to store it that way; API returns raw Base64).
- **Organizer scan:** read payload from QR or use stored **`qr_hash_hex`** → `POST /tickets/use` with `{ "qr_hash_hex": "..." }`.
- **Push notifications:** this backend sends **email** via the notifications queue only; for **FCM/APNs**, implement on the client and optionally add your own gateway later.

---

## Swagger UI

- **URL:** `http://localhost:{PORT}/api/v1/swagger/index.html` (default: port **8080**).

Click **Authorize**, enter:

```text
Bearer <paste_access_token_here>
```

(include the word `Bearer` and a space before the token). Then call secured endpoints from Swagger.

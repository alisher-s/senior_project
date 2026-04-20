# Student Event Ticketing Platform — Backend API

## Quick start (local development)

### Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (includes Docker Compose)
- That's it — no Go, Postgres, or Redis installation needed locally.

### Steps

1. **Clone the repository**

```bash
git clone <your-repo-url> senior_project
cd senior_project
```

2. **Create your `.env` file**

```bash
cp .env.example .env
```

Open `.env` and fill in your credentials (see [What each person needs](#what-each-person-needs) below).

3. **Start the stack**

```bash
docker compose up --build
```

This builds the API image and starts **api** (port **8080**), **postgres**, and **redis**.

4. **Verify the API**

```bash
curl -sS http://localhost:8080/api/v1/healthz
```

Expected: `{"status":"ok"}`.

5. **Swagger UI (interactive docs)**

Open: **http://localhost:8080/api/v1/swagger/index.html**

6. **Stop the stack**

```bash
docker compose down
```

To also wipe the database volume: `docker compose down -v`

---

## What each person needs

Everyone who runs this project locally needs their **own** credentials in their `.env`. **Never share your personal keys.**

### Email (SMTP) — optional

Emails are only used for ticket confirmation and event update notifications. The app runs fine without them (uses a no-op sender).

**Option A — Gmail App Password (recommended, uses your own Gmail):**
1. Enable 2-Step Verification on your Google account
2. Go to [myaccount.google.com](https://myaccount.google.com) → Security → App Passwords
3. Create an App Password for "Mail"
4. In `.env`: set `SMTP_USERNAME` and `SMTP_FROM` to your Gmail, `SMTP_PASSWORD` to the 16-character App Password, `SMTP_HOST=smtp.gmail.com`, `SMTP_PORT=587`

**Option B — Mailtrap (free sandbox, emails never go to real inboxes):**
1. Sign up free at [mailtrap.io](https://mailtrap.io) → Email Testing → Inboxes → SMTP Settings
2. Copy the credentials into `.env`

### Stripe payments — optional

Payments are optional. Without Stripe configured, all payment endpoints return `501 Not Implemented` — the rest of the app (free events, ticketing, auth, admin) works normally.

To enable payments:

1. Copy `STRIPE_SECRET_KEY` from the shared `.env` (the `sk_test_…` value) — this can be shared for local testing.
2. Install the [Stripe CLI](https://stripe.com/docs/stripe-cli)
3. In one terminal, run:
   ```bash
   stripe listen --forward-to localhost:8080/api/v1/payments/stripe/webhook
   ```
4. Copy the `whsec_…` key it prints → paste it as `STRIPE_WEBHOOK_SECRET` in your `.env`
   > **This key is session-specific** — you must run `stripe listen` yourself every time. You cannot reuse someone else's `whsec_…`.
5. Restart: `docker compose up --build`

Test card: **`4242 4242 4242 4242`**, any future expiry, any CVC.

> **Default currency is KZT.** Create paid events with `"price_amount": 5000` = 5000 ₸ (KZT is zero-decimal — no sub-units). Minimum Stripe charge is 50 KZT.

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
| `SMTP_USERNAME` | No | *(same as `SMTP_FROM`)* | Auth username — use explicitly for Mailtrap / Gmail App Passwords. |
| `SMTP_FROM` | No | *(empty)* | From address for outbound mail. |
| `SMTP_PASSWORD` | No | *(empty)* | SMTP password / App Password. |

### Payments (Stripe)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PAYMENTS_WEBHOOK_SECRET` | Yes (unless `APP_ENV=development`) | *(dev default applied)* | Shared secret for **HMAC-SHA256** verification of `POST /payments/webhook` bodies (`X-Signature`). |
| `STRIPE_SECRET_KEY` | No | *(empty — payments disabled)* | Stripe secret key (`sk_test_…` or `sk_live_…`). When set, payments are live. |
| `STRIPE_WEBHOOK_SECRET` | No | *(empty)* | Stripe webhook signing secret (`whsec_…`). Get from `stripe listen` CLI or Stripe dashboard. |
| `STRIPE_SUCCESS_URL` | No | `http://localhost:8080/payment-success` | URL Stripe redirects to after successful payment. |
| `STRIPE_CANCEL_URL` | No | `http://localhost:8080/payment-cancel` | URL Stripe redirects to when user cancels payment. |

### Push notifications (Firebase)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `FIREBASE_SERVER_KEY` | No | *(empty — push disabled)* | FCM legacy server key for push notifications. |

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
| **Request body** | `title` (required, 3–120), `description` (max 2000), `cover_image_url` (optional), `starts_at` (RFC3339, required), `capacity_total` (required, 1–100000), `price_amount` (integer ≥ 0, default 0 = free), `price_currency` (3-letter ISO, default `KZT`). |
| **Success** | **201** — `EventDTO` |
| **Errors** | **401** / **403**; **400** `invalid_request`; **500** `internal_error`. |

**Free event (no payment needed):**
```bash
curl -sS -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <organizer_access_token>" \
  -d '{"title":"NU Hackathon","description":"Annual hackathon","starts_at":"2026-06-01T10:00:00Z","capacity_total":100}'
```

**Paid event (5000 KZT):**
```bash
curl -sS -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <organizer_access_token>" \
  -d '{"title":"NU Gala","starts_at":"2026-06-01T18:00:00Z","capacity_total":50,"price_amount":5000,"price_currency":"KZT"}'
```
> **KZT is a zero-decimal currency** — `price_amount` is in tenge directly (`5000` = 5000 ₸). No sub-units. Minimum for Stripe is 50 KZT.

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
| **Errors** | **400**; **404** `not_found`; **402** `payment_required` (paid event — use `/payments/initiate`); **409** `capacity_full`, `already_registered`, `event_not_approved`, `event_not_published`, `event_cancelled`, `registration_closed`, etc. |

> **Only for free events** (`price_amount = 0`). For paid events use `POST /payments/initiate` — the ticket is created automatically after successful payment.

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
| **Errors** | **404** `ticket_not_found`; **409** `ticket_already_used`, `check_in_not_open`, `ticket_cannot_be_used`, etc. |

---

### Payment endpoints

Payments are powered by **Stripe** (test mode). Set `STRIPE_SECRET_KEY` in `.env` to enable. Without it, all payment endpoints return **501 Not Implemented**.

**Full flow for paid events:**
1. Call `POST /payments/initiate` → get `provider_url` (Stripe Checkout page)
2. Open `provider_url` in browser → enter test card `4242 4242 4242 4242`, any expiry/CVC
3. Stripe calls `POST /payments/stripe/webhook` → server **automatically creates the ticket**
4. User retrieves ticket + QR via `GET /tickets/my`

#### `POST /api/v1/payments/initiate`

| | |
|--|--|
| **Auth** | Yes |
| **Roles** | `student`, `organizer`, `admin` |
| **Request body** | `event_id` (string UUID, required). Amount and currency are read from the event — clients cannot override them. |
| **Success** | **201** — `payment_id`, `provider_ref`, `provider_url` (open this URL in browser to pay), `amount`, `currency` |
| **Errors** | **400** `free_event` (use `/tickets/register` instead); **404** `not_found`; **501** `not_implemented` (Stripe not configured); **401**; **500**. |

```bash
curl -sS -X POST http://localhost:8080/api/v1/payments/initiate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <student_access_token>" \
  -d '{"event_id":"550e8400-e29b-41d4-a716-446655440000"}'
```

Response:
```json
{
  "payment_id": "uuid",
  "provider_ref": "cs_test_...",
  "provider_url": "https://checkout.stripe.com/c/pay/cs_test_...",
  "amount": 5000,
  "currency": "KZT"
}
```

Open `provider_url` → pay → ticket is auto-created by webhook.

#### `POST /api/v1/payments/stripe/webhook`

Stripe-to-server webhook. **Do not call directly.** Verified using `Stripe-Signature` header and `STRIPE_WEBHOOK_SECRET`.

Handles:
- `checkout.session.completed` with `payment_status=paid` → creates ticket (idempotent)
- `checkout.session.expired` → marks payment canceled

For local testing, use Stripe CLI:
```bash
stripe listen --forward-to localhost:8080/api/v1/payments/stripe/webhook
```

#### `POST /api/v1/payments/webhook`

Generic HMAC webhook for non-Stripe providers.

| | |
|--|--|
| **Auth** | No |
| **Headers** | **`X-Signature`**: hex HMAC-SHA256 of raw body using `PAYMENTS_WEBHOOK_SECRET`. |
| **Request body** | `provider_ref` (string, required), `status` (`pending\|succeeded\|failed\|canceled`). |

```bash
BODY='{"provider_ref":"ref-123","status":"succeeded"}'
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "dev_webhook_secret_12345" -binary | xxd -p -c 256)
curl -sS -X POST http://localhost:8080/api/v1/payments/webhook \
  -H "Content-Type: application/json" -H "X-Signature: $SIG" -d "$BODY"
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
  "moderation_status": "pending | approved | rejected",
  "price_amount": 0,
  "price_currency": "KZT",
  "is_free": true
}
```

- `price_amount`: in the smallest currency unit. KZT (the default) is zero-decimal — no sub-units, so `5000` = 5000 ₸. `0` = free event.
- `is_free`: convenience boolean — `true` when `price_amount == 0`.
- `cover_image_url`: omitted when not set.

### Ticket (register / cancel / use responses; list item shapes differ slightly)

```json
{
  "ticket_id": "string (UUID)",
  "event_id": "string (UUID)",
  "user_id": "string (UUID)",
  "status": "active | used | cancelled",
  "qr_png_base64": "string (standard Base64 PNG, register only)",
  "qr_hash_hex": "string (hex)"
}
```

**`GET /tickets/my` item:** `ticket_id`, `status`, `qr_hash_hex`, `event_id`, `event_title`, `event_date` (RFC3339 string). No `qr_png_base64` in list.

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
| **402** | `payment_required` | Paid event — use `POST /payments/initiate` instead of `/tickets/register`. |
| **409** | `email_exists`, `already_registered`, `capacity_full`, … | Business conflicts (see handlers). |
| **400** | `free_event` | Called `/payments/initiate` for a free event — use `/tickets/register`. |
| **429** | `rate_limited` | Too many requests; check `Retry-After`. |
| **501** | `not_implemented` | Stripe not configured (`STRIPE_SECRET_KEY` not set). |

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

Current ticket **`status`** values in the API: **`active`**, **`used`**, **`cancelled`**.

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
              +--------------+---------------+
              |                              |
              v                              v
    +------------------+           +------------------+
    | POST /tickets/use|           | POST .../cancel  |
    | (organizer/admin)|           | (student)        |
    +--------+---------+           +--------+---------+
             |                                |
             v                                v
      +-------------+                  +-------------+
      |    used     |                  | cancelled   |
      +-------------+                  +-------------+
```

- **`active`:** valid ticket; show QR (`qr_png_base64` / `qr_hash_hex` from registration).
- **`used`:** check-in completed; show as “used” / hide QR for re-entry per product rules.
- **`cancelled`:** show as cancelled.

**Paid event flow:** payment success auto-creates the ticket (no separate register step needed). The ticket enters the same `active → used | cancelled` lifecycle as free tickets. Students retrieve their QR via `GET /tickets/my` after paying.

---

## Running without Docker

Requirements: **Go 1.22+**, **PostgreSQL**, **Redis** (same schema/migrations as Docker).

1. Apply migrations (see `docker/postgres/migrations` or your project script).
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

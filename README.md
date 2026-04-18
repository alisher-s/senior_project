# Student Event Ticketing Platform (NU) — backend

Модульный монолит на Go: домены `auth`, `events`, `ticketing`, `payments`, `notifications`, `admin`, `analytics`. Маршруты HTTP монтируются в **`internal/app/app.go`** под префиксом **`/api/v1`** (отдельного пакета `internal/api/v1` в репозитории нет; контракты описаны в Swagger и ниже).

---

## Базовый URL, порты и префикс API

| Сервис | Адрес (локально) | Примечание |
|--------|------------------|------------|
| **HTTP API** | **`http://localhost:8080`** | Префикс маршрутов: **`/api/v1`** |
| PostgreSQL (Docker, основной маппинг) | `localhost:5432` | `5432:5432` в `docker-compose.yml` |
| **PostgreSQL (хост, для локальных тестов / `psql`, если `:5432` занят)** | **`localhost:5433`** | Доп. маппинг **`5433:5432`** на тот же контейнер |
| Redis | `localhost:6379` | Rate limiting |

Пример: проверка живости — `GET /api/v1/healthz`.

**Мобильное приложение на реальном устройстве:** подставьте IP вашей машины в локальной сети вместо `localhost` (например `http://192.168.1.10:8080`), при условии что API запущен и порт `8080` доступен с телефона.

**CORS:** для локальной разработки фронта разрешены origin `http://localhost:3000` и `http://localhost:5173` (см. `internal/infra/http/middleware.go`). Другие origin по-прежнему не проходят без доработки списка.

---

## Роли и права (MVP P2)

| Роль | Назначение | Регистрация / просмотр | Создание событий / сканирование QR | Модерация / аналитика |
|------|------------|------------------------|-------------------------------------|-------------------------|
| **student** | Участник | `POST /auth/register`, `POST /auth/login`, список и карточка **одобренных** событий, `POST /tickets/register`, отмена своего билета | — | — |
| **organizer** | Организатор | Как student | `POST /events`, `PUT`/`DELETE` **своих** событий, `POST /tickets/use` (check-in по `qr_hash_hex`), `GET /analytics/events/stats` (только свои события) | — |
| **admin** | Администратор | Как organizer (в т.ч. события и check-in) | Без ограничения «только свои» для правок событий | `POST /admin/events/{id}/moderate`, `PATCH /admin/users/{id}/role`, `GET /admin/moderation-logs`, `GET /analytics/events/stats` (любые события) |

Защита маршрутов: JWT в заголовке `Authorization: Bearer <access_token>`. **401 Unauthorized** — нет или невалидный JWT (`missing_authorization`, `invalid_authorization`, `invalid_token`, `invalid_token_claims`, `invalid_credentials`, …). **403 Forbidden** — токен принят, но роль или правило не позволяют операцию (`forbidden`, `organizer_request_forbidden`, …). **409 Conflict** — конфликт доменной логики (например `already_registered`, `capacity_full` — см. таблицу ниже).

Публичная регистрация выдаёт только роль **`student`**. Учётки staff для dev: см. миграцию `006_dev_staff_users.sql` и раздел ниже.

---

## Аутентификация

- **Тип:** JWT — `Authorization: Bearer <access_token>`.
- **Пара access / refresh:** после `register` / `login` приходят `access_token` и `refresh_token`; refresh — для `POST /api/v1/auth/refresh` (тело: `{"refresh_token":"..."}`).
- **TTL (по умолчанию):** access — 15 минут, refresh — 30 суток (`JWT_ACCESS_TTL`, `JWT_REFRESH_TTL` в `config/.env.example`).
- **Регистрация:** `POST /api/v1/auth/register` — email в домене `AUTH_NU_EMAIL_DOMAIN` (по умолчанию `nu.edu.kz`). Новым пользователям назначается роль **`student`**.
- **Роли в токене:** строки `student`, `organizer`, `admin` (см. `user.role` в ответе auth).

**Как получить `organizer` / `admin` (не через register):**

1. **Сид в миграциях:** `docker/postgres/migrations/006_dev_staff_users.sql` — `staff.organizer@nu.edu.kz`, `staff.admin@nu.edu.kz`, пароль **`DevStaffPass1!`**.
2. **Админский API:** `PATCH /api/v1/admin/users/{id}/role` с телом `{"role":"organizer"|"admin"|"student"}`. После смены роли **отзываются все refresh-токены** — нужен повторный `login`.

---

## Формат тел запросов, дат и ошибок

- Тело запросов: **`Content-Type: application/json`**. Неизвестные поля JSON в ряде хендлеров отклоняются (`DisallowUnknownFields` / общий декодер).
- **Даты событий:** поле **`starts_at`** в формате **RFC3339** (например `2026-01-01T10:00:00Z`).
- **Обложка события (изображение):** опциональное поле **`cover_image_url`** — строка с **HTTPS URL** картинки. Сам файл на сервер API не загружается: хранится только ссылка (облако, CDN и т.п.), длина до **2048** символов. Задать можно при **`POST /api/v1/events`**, изменить при **`PUT /api/v1/events/{id}`**; чтобы **снять** обложку, в **PUT** передайте **`"cover_image_url": ""`**. В ответах поле **опускается**, если пустое (`omitempty`). Схема: миграция **`008_event_cover_image.sql`**.
- Успешные ответы — JSON; структуры полей см. Swagger (`/api/v1/swagger/index.html`).

### Стандартное тело ошибки

```json
{
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

### HTTP-статусы и типичные `error.code`

| HTTP | Когда | Примеры `error.code` |
|------|--------|----------------------|
| **400** | Невалидное тело/параметры | `invalid_request`, `invalid_id`, `email_not_allowed`, `invalid_role`, `invalid_action` |
| **401** | Нет или неверный JWT / подпись webhook | `missing_authorization`, `invalid_token`, `invalid_credentials`, `invalid_refresh_token`, `missing_signature`, … |
| **403** | JWT ок, роль не разрешена; запрос роли организатора не от студента; неверная подпись webhook | `forbidden`, `organizer_request_forbidden`, `invalid_signature` |
| **404** | Сущность не найдена (в т.ч. скрытые неодобренные события для публичного GET) | `not_found`, `ticket_not_found` |
| **409** | Конфликт бизнес-правил (билеты, вместимость, состояние события/билета) | `already_registered`, `capacity_full`, `event_not_approved`, `event_not_published`, `event_cancelled`, `registration_closed`, `ticket_already_used`, `organizer_already_active`, … |
| **429** | Rate limit | `rate_limited` |
| **501** | Функция ещё не реализована | `not_implemented` |
| **500** | Внутренняя ошибка | `internal_error` |

### Справочник `error.code` (текущая реализация)

**Auth:** `invalid_request`, `email_not_allowed`, `email_exists`, `invalid_credentials`, `invalid_refresh_token`, `refresh_token_consumed`, `organizer_already_active`, `organizer_request_forbidden`, `internal_error`.

**JWT / RBAC (middleware):** `missing_authorization`, `invalid_authorization`, `invalid_token`, `invalid_token_claims`, `missing_role`, `forbidden`.

**Общие:** `unauthorized`, `invalid_id`, `not_found`, `invalid_request`, `invalid_role`, `invalid_action`, `internal_error`, `not_implemented`, `rate_limited`.

**Билеты:** `capacity_full`, `already_registered`, `event_not_published`, `event_not_approved`, `event_cancelled`, `registration_closed`, `cancellation_not_allowed`, `check_in_not_open`, `ticket_not_found`, `ticket_already_cancelled`, `ticket_already_used`, `ticket_cannot_be_used`.

**Платежи:** `not_implemented`, `not_found` (webhook: неизвестный `provider_ref`), `internal_error`.

**Платежи (webhook):** `missing_signature`, `invalid_signature`.

---

## Регистрация билета: ответ

`POST /api/v1/tickets/register` (роль **student**), тело: `{"event_id":"<uuid>"}`.

Успешный ответ **201** включает:

| Поле | Тип | Описание |
|------|-----|----------|
| `ticket_id` | string | UUID билета |
| `event_id` | string | UUID события |
| `user_id` | string | UUID пользователя |
| `status` | string | Статус билета |
| **`qr_hash_hex`** | string | Хеш для check-in (`POST /tickets/use`) |
| **`qr_png_base64`** | string | Двоичный PNG, закодированный в **стандартный Base64** (без префикса `data:image/...`; при необходимости префикс добавляет клиент) |

Повторная регистрация на то же событие тем же пользователем: ожидайте **409** с `code: already_registered` (см. smoke-тесты в `agents.md`).

---

## Таблица эндпоинтов

Все пути относительно `http://<host>:8080/api/v1`.

| Метод | Путь | Auth | Роль | Назначение |
|--------|------|------|------|------------|
| GET | `/healthz` | Нет | — | Health check |
| GET | `/swagger/*` | Нет | — | Swagger UI |
| POST | `/auth/register` | Нет | — | Регистрация (`email`, `password` ≥ 8, домен NU) |
| POST | `/auth/login` | Нет | — | Вход |
| POST | `/auth/refresh` | Нет | — | Обновление пары токенов |
| PATCH | `/auth/me/roles` | Bearer | `student` (для запроса роли организатора) | Запрос роли organizer: тело `{"roles":["organizer"]}`; см. коды `organizer_request_forbidden`, `organizer_already_active` |
| POST | `/events` | Bearer | `organizer`, `admin` | Создание события; опционально **`cover_image_url`**; стартовая **модерация** — `pending` |
| GET | `/events` | Нет | — | Список **только одобренных** (`moderation_status=approved`); query: `limit` (по умолчанию **20**), `offset`, `q`, `starts_after`, `starts_before` (даты **RFC3339**) |
| GET | `/events/{id}` | Нет | — | Карточка **только для одобренного** события; иначе 404 |
| PUT | `/events/{id}` | Bearer | `organizer`, `admin` | Обновление полей, в т.ч. **`cover_image_url`** и `status`: `draft` / `published` / `cancelled`. Организатор — только **свои** события; иначе **403** |
| DELETE | `/events/{id}` | Bearer | `organizer`, `admin` | Удаление; то же правило владения для organizer |
| POST | `/tickets/register` | Bearer | `student` | Регистрация; см. поля QR выше |
| POST | `/tickets/{id}/cancel` | Bearer | `student` | Отмена своего билета |
| POST | `/tickets/use` | Bearer | `organizer`, `admin` | Вход по `qr_hash_hex` |
| POST | `/payments/initiate` | Bearer | `student`, `organizer`, `admin` | Старт оплаты (`event_id`, `amount`, `currency` — 3 буквы) |
| POST | `/payments/webhook` | Подпись `X-Signature` | — | Webhook провайдера (не для браузера) |
| POST | `/notifications/send-email` | Нет | — | Постановка письма в очередь (`to`, `title`, `body`) |
| GET | `/analytics/events/stats` | Bearer | **`organizer`, `admin`** | Метрики регистраций и вместимости; query `event_id` опционален. Organizer видит только свои события; чужое — **403** |
| PATCH | `/admin/users/{id}/role` | Bearer | `admin` | Назначение роли |
| POST | `/admin/events/{id}/moderate` | Bearer | `admin` | Модерация: тело `{"action":"approve"|"reject","reason":"..."}`; ответ `moderation_status` |
| GET | `/admin/moderation-logs` | Bearer | `admin` | Аудит модерации; query: `event_id`, `admin_id`, `limit`, `offset` |

Ответы `register` / `login` / `refresh` (`AuthResponseDTO`): `access_token`, `refresh_token`, `user`: `id`, `email`, `role`.

Событие в JSON содержит **`moderation_status`**: `pending` | `approved` | `rejected`. При заданной обложке в ответе присутствует **`cover_image_url`**.

---

## Swagger

После запуска API: **`/api/v1/swagger/index.html`**. Перегенерация из корня репозитория:

```bash
$(go env GOPATH)/bin/swag init -g cmd/api/main.go -o docs
```

---

## Rate limit

Redis: по умолчанию **`RATE_LIMIT_REQUESTS=120`** за **`RATE_LIMIT_WINDOW_SECONDS=60`**. При превышении — **429** и `code: rate_limited` (заголовок `Retry-After`).

---

## Состояние модулей (ожидания для клиентов)

| Модуль | Статус |
|--------|--------|
| **auth** | Register / login / refresh, JWT |
| **events** | CRUD; опциональная **обложка** по полю `cover_image_url` (HTTPS URL); создание только organizer/admin; публичный список и GET — только **одобренные** события; модерация через admin API |
| **ticketing** | Регистрация, QR, отмена; check-in — organizer/admin |
| **payments** | Заглушка; возможны **501** |
| **notifications** | Очередь и worker; HTTP может отвечать **501** |
| **admin** | Смена ролей и **модерация событий** |
| **analytics** | `GET /analytics/events/stats` для **organizer** и **admin** (реальные данные из БД) |

---

## Запуск локально

### Docker (рекомендуется, в том числе для фронтенда)

Из **корня** репозитория (где лежит `docker-compose.yml`):

```bash
docker compose up --build
```

Кратко для фронтенда:

1. Убедитесь, что установлен Docker (на macOS удобно OrbStack / Docker Desktop).
2. Выполните команду выше — поднимутся **api** на **`http://localhost:8080`**, **postgres** на хосте **`localhost:5432`** и **`localhost:5433`**, **redis** на **`6379`**.
3. Базовый URL API для запросов: `http://localhost:8080/api/v1/...`. Swagger UI: `http://localhost:8080/api/v1/swagger/index.html`.
4. Во фронте задайте базовый URL (например `VITE_API_URL=http://localhost:8080`). Origin dev-сервера должен быть **`http://localhost:3000`** или **`http://localhost:5173`** (CORS на бэкенде), либо расширьте список в `internal/infra/http/middleware.go`.

При **первом** создании тома Postgres выполняются миграции из `docker/postgres/migrations/`. Если меняли SQL после того, как том уже создан, см. `agents.md` (пересоздание тома или `scripts/apply-migrations.sh`).

### Только Go (нужны Postgres и Redis)

Если используете Postgres с хоста на порту **5433**, задайте `POSTGRES_PORT=5433` (и хост `localhost`) в `.env`. Скопируйте `config/.env.example` в `.env`, затем:

```bash
go run ./cmd/api
```

---

## Быстрые проверки (curl)

Регистрация студента:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"student@nu.edu.kz","password":"verystrongpassword"}'
```

Вход:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"student@nu.edu.kz","password":"verystrongpassword"}'
```

Создание события (нужен токен **organizer** или **admin**):

```bash
curl -sS -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <organizer_or_admin_access_token>" \
  -d '{"title":"NU Hackathon","description":"test event","starts_at":"2026-01-01T10:00:00Z","capacity_total":100,"cover_image_url":"https://example.com/covers/hackathon.jpg"}'
```

Одобрение события админом:

```bash
curl -sS -X POST "http://localhost:8080/api/v1/admin/events/<event_uuid>/moderate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_access_token>" \
  -d '{"action":"approve"}'
```

Регистрация билета:

```bash
curl -sS -X POST http://localhost:8080/api/v1/tickets/register \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <student_access_token>" \
  -d '{"event_id":"<uuid>"}'
```

Вход как staff-организатор и отметка по QR:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"staff.organizer@nu.edu.kz","password":"DevStaffPass1!"}'

curl -sS -X POST http://localhost:8080/api/v1/tickets/use \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <organizer_access_token>" \
  -d '{"qr_hash_hex":"<hex_from_register_response>"}'
```

---

## Стек (кратко)

- Go 1.22+
- PostgreSQL, Redis
- Chi, JWT (access + refresh в БД)
- Swagger (swag)

Вопросы по контрактам — через **Swagger** и этот файл; при изменении маршрутов обновляйте аннотации и команду `swag init` выше.

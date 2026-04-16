# Student Event Ticketing Platform (NU) — backend

Модульный монолит на Go: домены `auth`, `events`, `ticketing`, `payments`, `notifications`, `admin`, `analytics`. Этот документ описывает **текущее состояние API** для команды **мобильной разработки** и **фронтенда**.

---

## Базовый URL и префикс API

| Окружение | Base URL |
|-----------|----------|
| Docker / локально (по умолчанию) | `http://localhost:8080` |
| API | Все маршруты ниже — с префиксом **`/api/v1`** |

Пример: проверка живости сервиса — `GET /api/v1/healthz`.

**Мобильное приложение на реальном устройстве:** подставьте IP вашей машины в локальной сети вместо `localhost` (например `http://192.168.1.10:8080`), при условии что API запущен и порт `8080` доступен с телефона.

**Веб-фронтенд:** в репозитории **пока нет CORS middleware**. Запросы из браузера с другого origin могут блокироваться; типичные варианты: прокси в dev-сервере фронта или добавление CORS на бэкенде позже.

---

## Аутентификация

- **Тип:** JWT — в заголовке `Authorization: Bearer <access_token>`.
- **Пара access / refresh:** после `register` / `login` приходят `access_token` и `refresh_token`; access используется для защищённых методов, refresh — для `POST /api/v1/auth/refresh` (тело: `{"refresh_token":"..."}`).
- **TTL (по умолчанию):** access — 15 минут, refresh — 30 суток (см. `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL` в `config/.env.example`).
- **Регистрация:** `POST /api/v1/auth/register` принимает только email в домене из `AUTH_NU_EMAIL_DOMAIN` (по умолчанию `nu.edu.kz`). Новым пользователям назначается роль **`student`**.
- **Роли в токене:** строки `student`, `organizer`, `admin` (см. контракт ответа `user.role`).

**Как получить `organizer` / `admin` (не через register):**

1. **Сид в миграциях (локально / CI):** файл `docker/postgres/migrations/006_dev_staff_users.sql` создаёт учётки `staff.organizer@nu.edu.kz` и `staff.admin@nu.edu.kz` с паролем **`DevStaffPass1!`** (общий для обоих). Миграции выполняются при **первом** создании тома Postgres; при уже существующей БД — пересоздайте том или примените SQL вручную.
2. **Админский API:** `PATCH /api/v1/admin/users/{id}/role` с телом `{"role":"organizer"}` или `"admin"` / `"student"`. Доступен только с JWT роли **`admin`**. После смены роли у пользователя **отзываются все refresh-токены** — нужен повторный `login` / `refresh` уже с новой ролью.

Публичная регистрация **никогда** не выдаёт `organizer`/`admin` — это сделано намеренно.

---

## Формат тел запросов и ошибок

- Тело запросов: **`Content-Type: application/json`**. Неизвестные поля JSON в ряде хендлеров отклоняются (`DisallowUnknownFields`).
- Даты событий: **`starts_at`** в формате **RFC3339** (пример: `2026-01-01T10:00:00Z`).
- Успешные ответы — JSON с полями, описанными в таблице ниже или в Swagger.
- Ошибки — JSON вида:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "..."
  }
}
```

---

## Таблица эндпоинтов

Все пути относительно `https://<host>/api/v1`.

| Метод | Путь | Auth | Роль | Назначение |
|--------|------|------|------|------------|
| GET | `/healthz` | Нет | — | Health check |
| GET | `/swagger/*` | Нет | — | Swagger UI (удобно для согласования контрактов) |
| POST | `/auth/register` | Нет | — | Регистрация (`email`, `password` ≥ 8 символов, домен NU) |
| POST | `/auth/login` | Нет | — | Вход |
| POST | `/auth/refresh` | Нет | — | Обновление пары токенов |
| POST | `/events/` | Нет | — | Создание события (`title`, `description`, `starts_at`, `capacity_total`) |
| GET | `/events/` | Нет | — | Список; query: `limit`, `offset`, `q`, `starts_after`, `starts_before` |
| GET | `/events/{id}` | Нет | — | Карточка события |
| PUT | `/events/{id}` | Нет | — | Обновление (опционально `status`: `draft` / `published` / `cancelled`) |
| DELETE | `/events/{id}` | Нет | — | Удаление |
| POST | `/tickets/register` | Bearer | `student` | Регистрация на событие (`event_id`); ответ содержит `qr_png_base64`, `qr_hash_hex` |
| POST | `/tickets/{id}/cancel` | Bearer | `student` | Отмена своего билета |
| POST | `/tickets/use` | Bearer | `organizer` или `admin` | Отметка входа по `qr_hash_hex` в теле |
| POST | `/payments/initiate` | Bearer | `student`, `organizer` или `admin` | Старт оплаты (`event_id`, `amount`, `currency` — 3 буквы) |
| POST | `/payments/webhook` | Нет (подпись) | — | Webhook провайдера: заголовок **`X-Signature`** — hex HMAC-SHA256 от **сырого** тела; секрет `PAYMENTS_WEBHOOK_SECRET`. Для мобилки/веба обычно не вызывается |
| POST | `/notifications/send-email` | Нет | — | Постановка письма в очередь (`to`, `title`, `body`); см. статус модуля ниже |
| GET | `/analytics/events/stats` | Bearer | Любая роль из токена | Статистика; query `event_id` опционален |
| PATCH | `/admin/users/{id}/role` | Bearer | `admin` | Назначение роли пользователю (`{"role":"student"|"organizer"|"admin"}`) |
| POST | `/admin/events/{id}/moderate` | Bearer | `admin` | Модерация события (`action` в теле) |

Ответы регистрации/логина (`AuthResponseDTO`): `access_token`, `refresh_token`, `user`: `id`, `email`, `role`.

---

## Swagger

После запуска API: откройте в браузере **`/api/v1/swagger/index.html`** (например `http://localhost:8080/api/v1/swagger/index.html`). Спецификация: `doc.json` по пути Swagger.

---

## Ограничение частоты запросов (rate limit)

Используется Redis. Параметры по умолчанию: **`RATE_LIMIT_REQUESTS=120`** за **`RATE_LIMIT_WINDOW_SECONDS=60`** (см. `docker-compose.yml` / `.env.example`). При превышении клиент получит ошибку со стороны middleware.

---

## Состояние модулей (что ожидать при интеграции)

| Модуль | Статус для клиентов |
|--------|---------------------|
| **auth** | Рабочий сценарий: register / login / refresh, JWT |
| **events** | CRUD без авторизации (все могут создавать/менять — текущая модель «foundation») |
| **ticketing** | Регистрация, QR, отмена; скан — только `organizer`/`admin` |
| **payments** | Инициация и webhook заложены; провайдер и продуктовая логика могут быть упрощены — уточняйте у бэкенда перед продакшен-сценарием |
| **notifications** | Очередь и worker есть; HTTP `send-email` может отвечать **501 Not Implemented**, если отправка ещё не подключена — смотрите тело ошибки |
| **admin** | Смена роли пользователя (`PATCH .../users/{id}/role`); модерация событий — заглушка (может отвечать **501**) |
| **analytics** | Может отвечать **501**, пока аналитика не реализована |

---

## Запуск локально

### Docker (рекомендуется)

```bash
docker compose up --build
```

При **первом** старте PostgreSQL выполняются SQL-файлы из **`docker/postgres/migrations/`** (смонтированы в `docker-entrypoint-initdb.d`). Redis поднимается для rate limiting.

Секреты для разработки заданы в `docker-compose.yml`; для продакшена их нужно переопределять.

### Только Go (нужны Postgres и Redis)

Скопируйте `config/.env.example` в `.env` в корне или экспортируйте переменные, затем:

```bash
go run ./cmd/api
```

Обязательные переменные (вне `development`): как минимум `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`, `PAYMENTS_WEBHOOK_SECRET` — см. `internal/config/config.go`.

---

## Быстрые проверки (curl)

Регистрация:

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

Создание события:

```bash
curl -sS -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{"title":"NU Hackathon","description":"test event","starts_at":"2026-01-01T10:00:00Z","capacity_total":1}'
```

Регистрация билета (подставьте токен и `event_id`):

```bash
curl -sS -X POST http://localhost:8080/api/v1/tickets/register \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{"event_id":"<uuid>"}'
```

Вход как staff-организатор (после применения миграции `006_dev_staff_users.sql`) и отметка входа по QR:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"staff.organizer@nu.edu.kz","password":"DevStaffPass1!"}'

curl -sS -X POST http://localhost:8080/api/v1/tickets/use \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <organizer_access_token>" \
  -d '{"qr_hash_hex":"<hex_from_register_response>"}'
```

Назначить пользователю роль `organizer` (нужен токен **admin** — например `staff.admin@nu.edu.kz`):

```bash
curl -sS -X PATCH "http://localhost:8080/api/v1/admin/users/<user_uuid>/role" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_access_token>" \
  -d '{"role":"organizer"}'
```

---

## Стек (кратко)

- Go 1.22+
- PostgreSQL, Redis
- Chi, JWT (access + refresh в БД, одноразовый refresh)
- Swagger (swag)

Вопросы по контрактам лучше согласовывать через **Swagger** и этот файл; при изменении маршрутов обновляйте аннотации и перегенерацию `docs/`.

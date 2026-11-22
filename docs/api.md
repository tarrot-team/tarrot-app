# API документация `tarrot-app`

## Порты сервисов

| Сервис | Порт по умолчанию | Назначение |
|---|---|---|
| `tarot-service` | `:8081` | Матрица судьбы (метод Ладини), 78 карт Таро, расклады, AI-разбор, история |
| `community-service` | `:8082` | Профили пользователей, форум, посты, комментарии |
| `realtime-service` | `:8083` | Комнаты чата и живые уведомления (SSE) |

## Общие правила
- **Формат:** JSON (`Content-Type: application/json; charset=utf-8`), кроме SSE-потоков (`text/event-stream`).
- **Авторизация:** заголовок `Authorization: Bearer <jwt>` (HS256 JWT через общий пакет `pkg/auth`, поддельный `X-User-ID` игнорируется).
- **Формат ошибок:** `{"error": "описание ошибки"}`.
- **Переменные окружения:**
  - `PORT` — порт сервиса.
  - `JWT_SECRET` — секрет для подписи и проверки JWT (общий для `auth-service` и всех сервисов).
  - `DATABASE_URL` — строка подключения PostgreSQL (если не задана, сервисы работают в памяти).
  - `GROQ_API_KEY` или `LLM_API_KEY` — ключ для LLM в `tarot-service` (без ключа работает `fallback`).
  - `LLM_BASE_URL` — базовый URL OpenAI-совместимого API (по умолчанию `https://api.groq.com/openai/v1`).
  - `LLM_MODEL` — модель LLM (по умолчанию `llama-3.3-70b-versatile`).

---

## 1. `tarot-service` (`:8081`)

### `GET /health`
Проверка работоспособности. Ответ `200 OK`: `{"status": "ok"}`.

### `GET /arcana`
Возвращает массив всех 22 старших арканов в марсельском порядке (1–22) с полями `number`, `name`, `short_desc`, `plus`, `minus`, `love`, `money`.

### `GET /arcana/{number}`
Возвращает один старший аркан по номеру `1..22` (`400` если не число, `404` если вне диапазона).

### `GET /deck`
Возвращает полную колоду из 78 карт (`major_1..major_22` и `minor_{suit}_{1..14}`, где `suit`: `wands`, `cups`, `swords`, `pentacles`).

### `GET /matrix?birth_date=YYYY-MM-DD`
Рассчитывает полную матрицу судьбы по методу Ладини:
- базовые точки `a..m`,
- 14 внутренних чакровых и родовых точек (`inner`),
- кармический хвост (`karmic_tail`),
- линию отношений и денег (`love_money`),
- предназначение (`purpose`),
- 64 точки энергий по годам (`years`),
- словарь описаний выпавших арканов (`arcana`).

### `POST /matrix/interpret`
Возвращает матрицу и текстовый разбор (от AI или `fallback`).

Запрос:
```json
{
  "birth_date": "1997-01-21",
  "focus": "general"
}
```
Допустимые значения `focus`: `general` (по умолчанию), `love`, `money`, `karmic`, `purpose`.

### `POST /spread`
Делает расклад Таро и возвращает трактовку.
- Если передан заголовок `X-User-ID`, расклад сохраняется в историю, а в ответе добавляются `id` и `created_at`.
- **Авто-режим** (`cards` не передан): `type` — `daily` (1 карта), `three_cards` (3 карты, по умолчанию), `love` (3 карты).
- **Ручной режим** (реальная колода): передаётся массив `cards` (до 10 уникальных карт), `type` может быть `daily`, `three_cards`, `love` или `custom`.

Пример (ручной ввод):
```json
{
  "type": "custom",
  "question": "Стоит ли менять работу?",
  "cards": [
    {"card_id": "major_1", "reversed": false, "position": "Сейчас"},
    {"card_id": "minor_pentacles_10", "reversed": true, "position": "Что мешает"}
  ]
}
```

### `GET /spreads?limit=20`
Требует `Authorization: Bearer <jwt>`. Возвращает сохранённые расклады пользователя (от новых к старым, `limit` от 1 до 100).

### `GET /spreads/{id}`
Требует `Authorization: Bearer <jwt>`. Возвращает сохранённый расклад **только его владельцу** (`401` без токена, `403` для чужого пользователя, `404` если не найден).

### `DELETE /spreads/{id}`
Требует `Authorization: Bearer <jwt>`. Удаляет расклад владельца (`204 No Content`, `403` если чужой, `404` если не найден).

---

## 2. `community-service` (`:8082`)

### `GET /health`
Ответ `200 OK`: `{"status": "ok"}`.

### `POST /profiles`
Требует `X-User-ID`. Создаёт профиль пользователя (`201 Created`).
```json
{
  "username": "tarot_lover",
  "bio": "Изучаю матрицу судьбы",
  "avatar_url": "https://example.com/avatar.png",
  "birth_date": "1999-12-29"
}
```

### `GET /profiles/me`
Требует `X-User-ID`. Возвращает профиль текущего пользователя.

### `GET /profiles/{id}`
Возвращает публичный профиль пользователя по ID.

### `PATCH /profiles/{id}`
Требует `X-User-ID` (должен совпадать с `{id}`, иначе `403 Forbidden`). Частично обновляет поля профиля.

---

## 3. `realtime-service` (`:8083`)

### `GET /health`
Ответ `200 OK`: `{"status": "ok"}`.

### `GET /rooms/{room}/stream`
Требует `X-User-ID`. Открывает SSE-поток (`text/event-stream`) сообщений комнаты в реальном времени.

### `POST /rooms/{room}/messages`
Требует `X-User-ID`. Отправляет сообщение в комнату (`201 Created`), сохраняет в историю и рассылает всем подписчикам комнаты.
```json
{
  "text": "Привет, чат!"
}
```

### `GET /rooms/{room}/messages?limit=20`
Возвращает последние сообщения комнаты (`limit` 1..50, в хронологическом порядке) и счётчик `online`.

### `GET /notifications/stream`
Требует `X-User-ID`. Открывает SSE-поток личных уведомлений пользователя.

### `POST /notifications`
Требует `Authorization: Bearer <jwt>` (отправитель `from_user_id` берётся из проверенного JWT). Доставляет пользователю живое уведомление (`201 Created`).
```json
{
  "to_user_id": "user-1",
  "text": "Оставил комментарий под твоим постом"
}
```

# Цифровой двойник локомотива

Стек: **Go**, **PostgreSQL**, **RabbitMQ**, **WebSocket**, фронт в **`frontend/`**, бэкенд в **`backend/`**, симулятор в **`simulators/`**, брокер в **`rabbitmq/`** (отдельный compose). Общая модель телеметрии: **`pkg/telemetry`**, общая схема очереди: **`pkg/rabbitmq`**.

Поток данных: **симулятор → RabbitMQ → бэкенд → фронт** (live по WebSocket `/ws/telemetry`).

## Структура репозитория

| Папка | Содержимое |
|--------|------------|
| `frontend/` | HTML / CSS / JS дашборда (`index.html`, статика по `/static/`) |
| `backend/cmd/server` | Точка входа API и раздача фронта |
| `backend/internal/` | API, БД, health, store, WebSocket |
| `simulators/cmd/simulator` | CLI: публикация JSON только в **RabbitMQ** |
| `rabbitmq/` | `docker-compose.yml` — только брокер (можно поднять отдельно от Postgres) |
| `pkg/rabbitmq` | Имена exchange/очереди и объявление топологии |
| `simulators/synth` | Генератор «датчиков» (PRNG + состояние) |
| `pkg/telemetry` | Общие типы `Sample` / `Alert` для бэка и симулятора |

## Быстрый старт

Запускать команды **из корня репозитория** (`hacknu/`), чтобы путь `./frontend` находился автоматически.

1. Postgres и RabbitMQ:

```bash
docker compose up -d
```

Только брокер (из каталога `rabbitmq/`):

```bash
docker compose -f rabbitmq/docker-compose.yml up -d
```

2. API:

```bash
export DATABASE_URL="postgres://hacknu:hacknu@localhost:5432/locomotive?sslmode=disable"
export HTTP_ADDR=":8080"
# опционально: RABBITMQ_DISABLE=1 — без consumer (остаётся только POST /api/v1/telemetry)
go run ./backend/cmd/server
```

3. Симулятор (в другом терминале) — публикует в **RabbitMQ** (`amqp://hacknu:hacknu@127.0.0.1:5672/` по умолчанию):

```bash
go run ./simulators/cmd/simulator -train LOC-DEMO-001
```

Ручная подача записи без очереди: **HTTP** `POST /api/v1/telemetry` (curl и т.д.).

4. Браузер: [http://127.0.0.1:8080/](http://127.0.0.1:8080/)

Если бинарь запускается из другой директории, укажите путь к фронту:

```bash
export FRONTEND_DIR=/полный/путь/к/hacknu/frontend
```

## Переменные окружения

| Переменная | Назначение |
|------------|------------|
| `DATABASE_URL` | PostgreSQL (см. `backend/internal/db/pool.go`) |
| `HTTP_ADDR` | Адрес прослушивания, по умолчанию `:8080` |
| `FRONTEND_DIR` | Каталог с `index.html` и статикой, по умолчанию `frontend` |
| `RABBITMQ_URL` | AMQP URL бэкенда-consumer, по умолчанию `amqp://hacknu:hacknu@127.0.0.1:5672/` |
| `RABBITMQ_DISABLE` | `1` — не поднимать consumer (если брокера нет) |

Управление RabbitMQ: [http://127.0.0.1:15672/](http://127.0.0.1:15672/) (логин/пароль `hacknu` / `hacknu` при запуске через compose из этого репо).

## Тесты

```bash
docker compose up -d
export DATABASE_URL="postgres://hacknu:hacknu@localhost:5432/locomotive?sslmode=disable"
go test ./... -count=1
```

Интеграционный тест в `backend/internal/store` без Postgres пропускается. `SKIP_INTEGRATION=1 go test ./...` — без интеграции.

Спецификация API: `openapi.yaml`.

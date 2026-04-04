# Цифровой двойник локомотива

Стек: **Go**, **PostgreSQL**, **RabbitMQ**, **WebSocket**, фронт в **`frontend/`**, бэкенд в **`backend/`**, normalizer в **`normalizer/`**, симулятор в **`simulators/`**. Общая модель телеметрии: **`pkg/telemetry`**, общая схема очереди: **`pkg/rabbitmq`**.

Поток данных: **симулятор → RabbitMQ(raw) → normalizer → RabbitMQ(normalized) → бэкенд → фронт** (live по WebSocket `/ws/telemetry`).

## Структура репозитория

| Папка | Содержимое |
|--------|------------|
| `frontend/` | HTML / CSS / JS дашборда (`index.html`, статика по `/static/`) |
| `backend/cmd/server` | Точка входа API и раздача фронта |
| `normalizer/cmd/normalizer` | Отдельный микросервис normalizer (отдельный binary) |
| `normalizer/internal/` | Config, consumer и stateful preprocessing логика |
| `backend/internal/` | API, БД, health, store, WebSocket |
| `simulators/cmd/simulator` | CLI: публикация JSON только в **RabbitMQ** |
| `pkg/rabbitmq` | Имена exchange/очередей (`raw/normalized/dlq`) и объявление топологии |
| `simulators/synth` | Генератор «датчиков» (PRNG + состояние) |
| `pkg/telemetry` | Общие типы `Sample` / `Alert` для бэка и симулятора |

## Быстрый старт

Запускать команды **из корня репозитория** (`hacknu/`), чтобы путь `./frontend` находился автоматически.

1. Поднять сервисы (RabbitMQ + Postgres + backend + normalizer):

```bash
docker compose up -d
```

2. Симулятор (в отдельном терминале) — публикует raw в **RabbitMQ**:

```bash
go run ./simulators/cmd/simulator -train LOC-DEMO-001
```

3. Браузер: [http://127.0.0.1:8080/](http://127.0.0.1:8080/)

4. Логи сервисов:

```bash
docker compose logs -f backend normalizer
```

Если нужно запускать без compose (локально процессами):

```bash
# терминал 1
export DATABASE_URL="postgres://hacknu:hacknu@localhost:5432/locomotive?sslmode=disable"
export RABBITMQ_URL="amqp://hacknu:hacknu@127.0.0.1:5672/"
export FRONTEND_DIR=/полный/путь/к/hacknu/frontend
go run ./backend/cmd/server

# терминал 2
export RABBITMQ_URL="amqp://hacknu:hacknu@127.0.0.1:5672/"
export NORMALIZER_ENABLE_SMOOTHING=true
export NORMALIZER_ENABLE_DEDUP=true
export NORMALIZER_DEDUP_WINDOW_MS=1500
export NORMALIZER_STATE_TTL_MIN=15
export NORMALIZER_BUFFER_SIZE=5
export NORMALIZER_EMA_ALPHA=0.4
go run ./normalizer/cmd/normalizer
```

## Переменные окружения

| Переменная | Назначение |
|------------|------------|
| `DATABASE_URL` | PostgreSQL (см. `backend/internal/db/pool.go`) |
| `HTTP_ADDR` | Адрес прослушивания, по умолчанию `:8080` |
| `FRONTEND_DIR` | Каталог с `index.html` и статикой, по умолчанию `frontend` |
| `RABBITMQ_URL` | AMQP URL бэкенда-consumer, по умолчанию `amqp://hacknu:hacknu@127.0.0.1:5672/` |
| `NORMALIZER_ENABLE_SMOOTHING` | Включить EMA сглаживание (`true/false`) |
| `NORMALIZER_ENABLE_DEDUP` | Включить дедупликацию (`true/false`) |
| `NORMALIZER_DEDUP_WINDOW_MS` | Окно дедупликации в миллисекундах |
| `NORMALIZER_STATE_TTL_MIN` | TTL train-state в минутах |
| `NORMALIZER_BUFFER_SIZE` | Размер буфера последних sample на поезд |
| `NORMALIZER_EMA_ALPHA` | Коэффициент EMA (0..1) |
| `RABBITMQ_DISABLE` | `1` — не поднимать consumer (если брокера нет) |

Управление RabbitMQ: [http://127.0.0.1:15672/](http://127.0.0.1:15672/) (логин/пароль `hacknu` / `hacknu` при запуске через compose из этого репо).

## Тесты

```bash
docker compose up -d
export DATABASE_URL="postgres://hacknu:hacknu@localhost:5432/locomotive?sslmode=disable"
go test ./... -count=1
```

Интеграционный тест в `backend/internal/store` без Postgres пропускается. `SKIP_INTEGRATION=1 go test ./...` — без интеграции.

Спецификация API: `openapi.yaml`

# Цифровой двойник локомотива

Стек: **Go**, **PostgreSQL**, **RabbitMQ**, **WebSocket**, фронт в **`frontend/`**, бэкенд в **`backend/`**, симулятор в **`simulators/`**, брокер в **`rabbitmq/`** (отдельный compose). Общая модель телеметрии: **`pkg/telemetry`**, общая схема очереди: **`pkg/rabbitmq`**.

Поток данных: **симулятор → RabbitMQ → бэкенд → фронт** (live по WebSocket `/ws/telemetry`).

## Структура репозитория

| Папка | Содержимое |
|--------|------------|
| `frontend/` | HTML / CSS / JS дашборда (`index.html`, статика по `/static/`) |
| `backend/cmd/server` | Точка входа API и раздача фронта |
| `backend/internal/` | API, БД, health, store, WebSocket, **llm** (ИИ-разбор по запросу) |
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
export DATABASE_URL="postgres://hacknu:hacknu@127.0.0.1:5433/locomotive?sslmode=disable"
export HTTP_ADDR=":8080"
# опционально: ИИ-разбор (Gemini или OpenAI)
# export GEMINI_API_KEY="AIzaSy..."
# export GEMINI_MODEL="gemini-2.0-flash"
# либо: export OPENAI_API_KEY="sk-..."
# На Windows: создайте `.env.local.ps1` (в .gitignore) с $env:GEMINI_API_KEY = "..." — его подхватывает `run.ps1`.
# опционально: RABBITMQ_DISABLE=1 — без consumer (остаётся только POST /api/v1/telemetry)
go run ./backend/cmd/server
```

На **Windows** удобно `.\run.ps1` (подставляет Go из `Program Files` и корректный `DATABASE_URL`, даже если в профиле остался старый `DATABASE_URL` на `localhost:5432`).

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
| `DATABASE_URL` | PostgreSQL (см. `backend/internal/db/pool.go`). Если переменная уже задана в системе на другой хост/порт, она **перекрывает** значение по умолчанию — задайте её явно или используйте `run.ps1`. |
| `HTTP_ADDR` | Адрес прослушивания, по умолчанию `:8080` |
| `FRONTEND_DIR` | Каталог с `index.html` и статикой, по умолчанию `frontend` |
| `RABBITMQ_URL` | AMQP URL бэкенда-consumer, по умолчанию `amqp://hacknu:hacknu@127.0.0.1:5672/` |
| `RABBITMQ_DISABLE` | `1` — не поднимать consumer (если брокера нет) |
| `GEMINI_API_KEY` | Ключ **Google AI** (Gemini), формат `AIzaSy…`. Удобно для демо. |
| `GEMINI_MODEL` | Модель Gemini (`generateContent`), по умолчанию `gemini-2.0-flash`. |
| `OPENAI_API_KEY` | Ключ **OpenAI** (`sk-…`). Если задан вместе с Gemini, приоритет у `GEMINI_API_KEY`. Ключи `AIza…`, ошибочно положенные в `OPENAI_API_KEY`, автоматически обрабатываются как Gemini. |
| `OPENAI_MODEL` | Модель OpenAI Responses API, по умолчанию `gpt-4o-mini`. |
| `OPENAI_BASE_URL` | Для **OpenRouter**: `https://openrouter.ai/api/v1`. Для ключей `sk-or-v1-…` база подставляется автоматически, если пусто. |
| `OPENAI_USE_CHAT` | `1` — принудительно **Chat Completions** вместо Responses API (нужно для OpenRouter и многих прокси). Ключи `sk-or-v1-…` включают chat сами. |
| Файл `.env` | В корне репо (см. `env.example`): подставляется при старте процесса, если переменная ещё не задана. Удобно при запуске из IDE без `run.ps1`. |
| `GOOGLE_MAPS_API_KEY` | Ключ [Maps JavaScript API](https://developers.google.com/maps/documentation/javascript) для блока «Карта» на дашборде (маркер по `lat`/`lon` из телеметрии). В Google Cloud ограничьте ключ по HTTP referrer и включите биллинг. Без ключа блок скрыт, подсказка в UI. |

Управление RabbitMQ: [http://127.0.0.1:15672/](http://127.0.0.1:15672/) (логин/пароль `hacknu` / `hacknu` при запуске через compose из этого репо).

## Тесты

```bash
docker compose up -d
export DATABASE_URL="postgres://hacknu:hacknu@127.0.0.1:5433/locomotive?sslmode=disable"
go test ./... -count=1
```

Интеграционный тест в `backend/internal/store` без Postgres пропускается. `SKIP_INTEGRATION=1 go test ./...` — без интеграции.

Спецификация API: `openapi.yaml`.

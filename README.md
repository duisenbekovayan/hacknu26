# Цифровой двойник локомотива

Стек: **Go**, **PostgreSQL**, **RabbitMQ**, **WebSocket**, фронт в **`frontend/`**, бэкенд в **`backend/`**, normalizer в **`normalizer/`**, симулятор в **`simulators/`**. Общая модель телеметрии: **`pkg/telemetry`**, общая схема очереди: **`pkg/rabbitmq`**.

Поток данных: **симулятор → RabbitMQ (`telemetry.raw`) → normalizer → RabbitMQ (`telemetry.normalized`) → бэкенд → фронт** (live по WebSocket `/ws/telemetry` и опрос REST).

Без **normalizer** сообщения из симулятора не доходят до бэкенда (остаются в `telemetry.raw`). Ручной **`POST /api/v1/telemetry`** обходит очереди и идёт сразу в API.

## Структура репозитория

| Папка | Содержимое |
|--------|------------|
| `frontend/` | HTML / CSS / JS дашборда (`index.html`, статика по `/static/`) |
| `backend/cmd/server` | Точка входа API и раздача фронта |
| `backend/internal/` | API, БД, health, store, WebSocket, **llm** (ИИ-разбор по запросу) |
| `normalizer/cmd/normalizer` | Отдельный микросервис normalizer (отдельный binary) |
| `normalizer/internal/` | Config, consumer и stateful preprocessing логика |
| `backend/internal/` | API, БД, health, store, WebSocket |
| `simulators/cmd/simulator` | CLI: публикация JSON только в **RabbitMQ** |
| `pkg/rabbitmq` | Имена exchange/очередей (`raw/normalized/dlq`) и объявление топологии |
| `simulators/synth` | Генератор «датчиков» (PRNG + состояние) |
| `pkg/telemetry` | Общие типы `Sample` / `Alert` для бэка и симулятора |

## Быстрый старт

Запускать команды **из корня репозитория** (`hacknu/`), чтобы путь `./frontend` находился автоматически.

1. Поднять всё (RabbitMQ, Postgres, **normalizer**, **simulator**, **backend** с фронтом):

```bash
docker compose up -d
```

Ключи **ИИ** и **Google Maps** положите в файл **`.env`** в корне репозитория (`GEMINI_API_KEY` / `OPENAI_API_KEY`, при необходимости `GOOGLE_MAPS_API_KEY`). Docker Compose подставляет их в сервис `backend`; без этого в контейнере ИИ и карта будут отключены, хотя при локальном `go run` всё работает. После смены `.env` перезапустите backend: `docker compose up -d --force-recreate backend`.

Симулятор в compose публикует raw в RabbitMQ (`train=LOC-DEMO-001`, интервал 1 с). Параметры можно сменить в `docker-compose.yml` (`command:` у сервиса `simulator`) или собрать образ с другим `CMD` в `simulators/Dockerfile`.

2. Браузер: [http://127.0.0.1:8080/](http://127.0.0.1:8080/) — фронт отдаёт **backend**.

3. Логи:

```bash
docker compose logs -f backend normalizer simulator
```

Локально без Docker-симулятора: `go run ./simulators/cmd/simulator -train LOC-DEMO-001`.

Если нужно запускать **без** полного compose (только инфра или всё кроме Go-сервисов), сначала поднимите RabbitMQ и Postgres:

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
docker compose up -d rabbitmq postgres
```

Дальше процессы **в таком порядке** (normalizer должен быть запущен до или вместе с симулятором, иначе raw-очередь не обработается):

```bash
# терминал 1 — normalizer
export RABBITMQ_URL="amqp://hacknu:hacknu@127.0.0.1:5672/"
export NORMALIZER_ENABLE_SMOOTHING=true
export NORMALIZER_ENABLE_DEDUP=true
export NORMALIZER_DEDUP_WINDOW_MS=1500
export NORMALIZER_STATE_TTL_MIN=15
export NORMALIZER_BUFFER_SIZE=5
export NORMALIZER_EMA_ALPHA=0.4
go run ./normalizer/cmd/normalizer

# терминал 2 — backend
export DATABASE_URL="postgres://hacknu:hacknu@localhost:5432/locomotive?sslmode=disable"
export RABBITMQ_URL="amqp://hacknu:hacknu@127.0.0.1:5672/"
export FRONTEND_DIR=/полный/путь/к/hacknu/frontend
go run ./backend/cmd/server

# терминал 3 — симулятор (после п.1–2)
go run ./simulators/cmd/simulator -train LOC-DEMO-001
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
| `NORMALIZER_ENABLE_SMOOTHING` | Включить EMA сглаживание (`true/false`) |
| `NORMALIZER_ENABLE_DEDUP` | Включить дедупликацию (`true/false`) |
| `NORMALIZER_DEDUP_WINDOW_MS` | Окно дедупликации в миллисекундах |
| `NORMALIZER_STATE_TTL_MIN` | TTL train-state в минутах |
| `NORMALIZER_BUFFER_SIZE` | Размер буфера последних sample на поезд |
| `NORMALIZER_EMA_ALPHA` | Коэффициент EMA (0..1) |
| `RABBITMQ_DISABLE` | `1` — отключить consumer бэкенда (тогда телеметрия только через `POST`; поток **симулятор → очередь** не работает) |

Управление RabbitMQ: [http://127.0.0.1:15672/](http://127.0.0.1:15672/) (логин/пароль `hacknu` / `hacknu` при запуске через compose из этого репо).

## Тесты

```bash
docker compose up -d
export DATABASE_URL="postgres://hacknu:hacknu@127.0.0.1:5433/locomotive?sslmode=disable"
go test ./... -count=1
```

Интеграционный тест в `backend/internal/store` без Postgres пропускается. `SKIP_INTEGRATION=1 go test ./...` — без интеграции.

Спецификация API: `openapi.yaml`.

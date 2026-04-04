# Цифровой двойник локомотива

Стек: **Go**, **PostgreSQL**, **WebSocket**, фронт в **`frontend/`**, бэкенд в **`backend/`**, симулятор в **`simulators/`**. Общая модель телеметрии: **`pkg/telemetry`**.

## Структура репозитория

| Папка | Содержимое |
|--------|------------|
| `frontend/` | HTML / CSS / JS дашборда (`index.html`, статика по `/static/`) |
| `backend/cmd/server` | Точка входа API и раздача фронта |
| `backend/internal/` | API, БД, health, store, WebSocket |
| `simulators/cmd/simulator` | CLI: отправка сгенерированной телеметрии на API |
| `simulators/synth` | Генератор «датчиков» (PRNG + состояние) |
| `pkg/telemetry` | Общие типы `Sample` / `Alert` для бэка и симулятора |

## Быстрый старт

Запускать команды **из корня репозитория** (`hacknu/`), чтобы путь `./frontend` находился автоматически.

1. Postgres:

```bash
docker compose up -d
```

2. API:

```bash
export DATABASE_URL="postgres://hacknu:hacknu@localhost:5432/locomotive?sslmode=disable"
export HTTP_ADDR=":8080"
go run ./backend/cmd/server
```

3. Симулятор (в другом терминале):

```bash
go run ./simulators/cmd/simulator -url http://127.0.0.1:8080 -train LOC-DEMO-001
```

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

## Тесты

```bash
docker compose up -d
export DATABASE_URL="postgres://hacknu:hacknu@localhost:5432/locomotive?sslmode=disable"
go test ./... -count=1
```

Интеграционный тест в `backend/internal/store` без Postgres пропускается. `SKIP_INTEGRATION=1 go test ./...` — без интеграции.

Спецификация API: `openapi.yaml`. План: `PLAN.md`.

# Цифровой двойник локомотива

Стек: **Go**, **PostgreSQL**, **RabbitMQ**, **WebSocket**, фронт в **`frontend/`**, бэкенд в **`backend/`**, normalizer в **`normalizer/`**, симулятор в **`simulators/`**. Общая модель телеметрии: **`pkg/telemetry`**, общая схема очереди: **`pkg/rabbitmq`**.

Поток данных: **симулятор → RabbitMQ (`telemetry.raw`) → normalizer → RabbitMQ (`telemetry.normalized`) → бэкенд → фронт** (live по WebSocket `/ws/telemetry` и опрос REST).

Без **normalizer** сообщения из симулятора не доходят до бэкенда (остаются в `telemetry.raw`). Ручной **`POST /api/v1/telemetry`** обходит очереди и идёт сразу в API.

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

1. Поднять всё (RabbitMQ, Postgres, **normalizer**, **simulator**, **backend** с фронтом):

```bash
docker compose up -d
```

Симулятор в compose публикует raw в RabbitMQ (`train=LOC-DEMO-001`, интервал 1 с). Параметры можно сменить в `docker-compose.yml` (`command:` у сервиса `simulator`) или собрать образ с другим `CMD` в `simulators/Dockerfile`.

2. Браузер: [http://127.0.0.1:8080/](http://127.0.0.1:8080/) — фронт отдаёт **backend**.

3. Логи:

```bash
docker compose logs -f backend normalizer simulator
```

Локально без Docker-симулятора: `go run ./simulators/cmd/simulator -train LOC-DEMO-001`.

Если нужно запускать **без** полного compose (только инфра или всё кроме Go-сервисов), сначала поднимите RabbitMQ и Postgres:

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
| `RABBITMQ_DISABLE` | `1` — отключить consumer бэкенда (тогда телеметрия только через `POST`; поток **симулятор → очередь** не работает) |

Управление RabbitMQ: [http://127.0.0.1:15672/](http://127.0.0.1:15672/) (логин/пароль `hacknu` / `hacknu` при запуске через compose из этого репо).

## Тесты

```bash
docker compose up -d
export DATABASE_URL="postgres://hacknu:hacknu@localhost:5432/locomotive?sslmode=disable"
go test ./... -count=1
```

Интеграционный тест в `backend/internal/store` без Postgres пропускается. `SKIP_INTEGRATION=1 go test ./...` — без интеграции.

Спецификация API: `openapi.yaml`.

## Индекс здоровья (прозрачная формула)

Расчёт в `backend/internal/health/calc.go` после каждого сэмпла (и при ingest из RabbitMQ).

1. Суммируются **штрафы** `pen` по правилам ниже (каждое срабатывание добавляет вес; несколько ТЭД могут дать несколько штрафов).
2. **Индекс**: `health_index = clamp(100 − pen, 0, 100)` (округление до 0.1).
3. **Грейд**: A ≥ 85, B ≥ 70, C ≥ 60, D ≥ 40, E — если ниже 40.
4. В ответ попадают **top‑5 факторов** по величине штрафа (`health_top_factors`).

| Условие | Штраф (вклад в `pen`) |
|--------|------------------------|
| ОЖ `coolant_temp_c` &gt; **98** °C | `(temp − 98) × 5` |
| Давление масла `engine_oil_pressure_bar` &lt; **3.2** бар | **15** |
| АКБ `battery_voltage_v` вне **100…128** В | **10** |
| ТЭД `traction_motor_temp_c[i]` &gt; **115** °C | **8** на каждый такой двигатель |
| Главный резервуар `main_reservoir_bar` &lt; **7.0** бар | **12** |
| Алерт `severity: warn` | **5** за код |
| Алерт `severity: crit` | **15** за код |

Пороги сейчас **зашиты в коде**; см. раздел ниже, как вынести их без перекомпиляции.

### Пороги без перекомпиляции (что это и что делать)

**Суть:** в ТЗ просят менять пороги индекса через конфиг/БД/API, а не только правкой Go.

**Варианты (от простого к «как в проде»):**

1. **Файл конфигурации** (YAML/JSON), путь через `HEALTH_THRESHOLDS_FILE`; при старте сервера читать в структуру и подставлять в расчёт — без правки исходников, достаточно перезапуска процесса.
2. **Таблица в PostgreSQL** + загрузка при старте или раз в N секунд; админка через `GET/PUT /api/v1/settings/thresholds` (с базовой авторизацией).
3. **Переменные окружения** для числовых порогов (быстрый компромисс для демо).

Для защиты достаточно кратко объяснить жюри выбранный вариант и показать пример JSON порогов в README или в ответе API.

## История для графиков и replay

`GET /api/v1/telemetry/history?train_id=…&minutes=15` — точки за последние 15 минут (до 120 минут, до 5000 точек), порядок **от старых к новым**. На дашборде: выбор окна 5/10/15 мин, режим **Replay** с ползунком и кнопкой воспроизведения; на графиках — zoom колесом и pan по оси X (плагин Chart.js + Hammer.js).

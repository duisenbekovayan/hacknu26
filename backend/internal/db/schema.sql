-- Телеметрия: сырой JSON + рассчитанный индекс для быстрых запросов
CREATE TABLE IF NOT EXISTS telemetry_events (
    id BIGSERIAL PRIMARY KEY,
    train_id TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    health_index NUMERIC(5, 1) NOT NULL,
    health_grade CHAR(1) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_telemetry_train_time
    ON telemetry_events (train_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_telemetry_recorded_at
    ON telemetry_events (recorded_at DESC);

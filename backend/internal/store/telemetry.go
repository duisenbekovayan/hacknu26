package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hacknu/pkg/telemetry"
)

// Telemetry persists samples to PostgreSQL.
type Telemetry struct {
	pool *pgxpool.Pool
}

func NewTelemetry(pool *pgxpool.Pool) *Telemetry {
	return &Telemetry{pool: pool}
}

// Insert сохраняет событие.
func (t *Telemetry) Insert(ctx context.Context, s *telemetry.Sample) error {
	payload, err := json.Marshal(s)
	if err != nil {
		return err
	}
	ts, err := s.ParsedTime()
	if err != nil {
		return fmt.Errorf("ts: %w", err)
	}
	_, err = t.pool.Exec(ctx, `
		INSERT INTO telemetry_events (train_id, recorded_at, payload, health_index, health_grade)
		VALUES ($1, $2, $3, $4, $5)
	`, s.TrainID, ts, payload, s.HealthIndex, s.HealthGrade)
	return err
}

// History возвращает последние limit записей по поезду.
func (t *Telemetry) History(ctx context.Context, trainID string, limit int) ([]telemetry.Sample, error) {
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	rows, err := t.pool.Query(ctx, `
		SELECT payload
		FROM telemetry_events
		WHERE train_id = $1
		ORDER BY recorded_at DESC
		LIMIT $2
	`, trainID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []telemetry.Sample
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var s telemetry.Sample
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Latest возвращает последнюю запись по поезду.
func (t *Telemetry) Latest(ctx context.Context, trainID string) (*telemetry.Sample, error) {
	row := t.pool.QueryRow(ctx, `
		SELECT payload
		FROM telemetry_events
		WHERE train_id = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`, trainID)
	var raw []byte
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var s telemetry.Sample
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Prune удаляет записи старше retention (для демо 72ч).
func (t *Telemetry) Prune(ctx context.Context, retention time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-retention)
	tag, err := t.pool.Exec(ctx, `DELETE FROM telemetry_events WHERE recorded_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

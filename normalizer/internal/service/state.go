package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"hacknu/pkg/telemetry"
)

type emaValue struct {
	Set   bool
	Value float64
}

func (e *emaValue) Apply(x, alpha float64) float64 {
	if !e.Set {
		e.Set = true
		e.Value = x
		return x
	}
	e.Value = alpha*x + (1-alpha)*e.Value
	return e.Value
}

type emaState struct {
	SpeedKmh             emaValue
	FuelRateLph          emaValue
	BrakePipePressureBar emaValue
	MainReservoirBar     emaValue
	EngineOilPressureBar emaValue
	CoolantTempC         emaValue
	EngineOilTempC       emaValue
	BatteryVoltageV      emaValue
	TractionCurrentA     emaValue
	LineVoltageV         emaValue
	TractionMotorTempC   []emaValue
}

// TrainState хранит состояние обработки по train_id.
type TrainState struct {
	Buffer            []telemetry.Sample
	EMA               emaState
	LastFingerprint   string
	LastFingerprintAt time.Time
	LastSeen          time.Time
}

// Store хранит state per train_id с TTL-cleanup.
type Store struct {
	mu         sync.Mutex
	trains     map[string]TrainState
	ttl        time.Duration
	bufferSize int
}

func NewStore(ttl time.Duration, bufferSize int) *Store {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if bufferSize <= 0 {
		bufferSize = 5
	}
	return &Store{
		trains:     make(map[string]TrainState),
		ttl:        ttl,
		bufferSize: bufferSize,
	}
}

func (s *Store) BufferSize() int {
	return s.bufferSize
}

func (s *Store) Snapshot(trainID string) TrainState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.trains[trainID]
	if !ok {
		return TrainState{}
	}
	return cloneTrainState(st)
}

func (s *Store) Commit(trainID string, st TrainState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trains[trainID] = cloneTrainState(st)
}

func (s *Store) CleanupExpired(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for trainID, st := range s.trains {
		if st.LastSeen.IsZero() {
			continue
		}
		if now.Sub(st.LastSeen) > s.ttl {
			delete(s.trains, trainID)
			removed++
		}
	}
	return removed
}

func (s *Store) RunCleanup(ctx context.Context, log *slog.Logger) {
	interval := s.ttl / 3
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			removed := s.CleanupExpired(now)
			if removed > 0 {
				log.Info("state cleanup", "removed", removed)
			}
		}
	}
}

func (s *Store) size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.trains)
}

func cloneTrainState(in TrainState) TrainState {
	out := in
	if len(in.Buffer) > 0 {
		out.Buffer = append([]telemetry.Sample(nil), in.Buffer...)
	} else {
		out.Buffer = nil
	}
	if len(in.EMA.TractionMotorTempC) > 0 {
		out.EMA.TractionMotorTempC = append([]emaValue(nil), in.EMA.TractionMotorTempC...)
	} else {
		out.EMA.TractionMotorTempC = nil
	}
	return out
}

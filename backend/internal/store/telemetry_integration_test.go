package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"hacknu/backend/internal/db"
	"hacknu/backend/internal/health"
	"hacknu/backend/internal/store"
	"hacknu/pkg/telemetry"
)

func TestTelemetry_insertLatestHistory(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("SKIP_INTEGRATION set")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx)
	if err != nil {
		t.Skipf("postgres: %v (запустите: docker compose up -d)", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	st := store.NewTelemetry(pool)
	trainID := "TEST-" + t.Name()

	s := &telemetry.Sample{
		TS:                   time.Now().UTC().Format(time.RFC3339),
		TrainID:              trainID,
		SpeedKmh:             50,
		FuelLevelL:           3000,
		FuelRateLph:          20,
		BrakePipePressureBar: 5.0,
		MainReservoirBar:     8.0,
		EngineOilPressureBar: 4.4,
		CoolantTempC:         90,
		EngineOilTempC:       92,
		TractionMotorTempC:   []float64{72, 72, 72, 72, 72, 72},
		BatteryVoltageV:      110,
		TractionCurrentA:     500,
		LineVoltageV:         2750,
		Lat:                  51.0,
		Lon:                  71.0,
		MileageKm:            1,
		Alerts:               nil,
	}
	health.Apply(s)

	if err := st.Insert(ctx, s); err != nil {
		t.Fatalf("insert: %v", err)
	}

	latest, err := st.Latest(ctx, trainID)
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest == nil {
		t.Fatal("latest is nil")
	}
	if latest.HealthIndex != s.HealthIndex || latest.SpeedKmh != 50 {
		t.Fatalf("latest mismatch: %+v", latest)
	}

	hist, err := st.History(ctx, trainID, 10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(hist) != 1 {
		t.Fatalf("history len %d", len(hist))
	}

	_, err = pool.Exec(ctx, `DELETE FROM telemetry_events WHERE train_id = $1`, trainID)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

package service

import (
	"errors"
	"math"
	"testing"
	"time"

	"hacknu/pkg/telemetry"
)

func TestNormalizeSample_TrainIDRequired(t *testing.T) {
	s := validSample()
	s.TrainID = "  "

	_, err := NormalizeSample(&s)
	if !errors.Is(err, ErrTrainIDMissing) {
		t.Fatalf("err=%v, want %v", err, ErrTrainIDMissing)
	}
}

func TestNormalizeSample_TSNormalization(t *testing.T) {
	t.Run("empty ts -> set now", func(t *testing.T) {
		s := validSample()
		s.TS = ""

		out, err := NormalizeSample(&s)
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if out.TS == "" {
			t.Fatal("ts must be set")
		}
		if _, err := time.Parse(time.RFC3339, out.TS); err != nil {
			t.Fatalf("ts not RFC3339: %v", err)
		}
	})

	t.Run("invalid ts", func(t *testing.T) {
		s := validSample()
		s.TS = "bad-ts"

		_, err := NormalizeSample(&s)
		if !errors.Is(err, ErrTSInvalid) {
			t.Fatalf("err=%v, want %v", err, ErrTSInvalid)
		}
	})
}

func TestNormalizeSample_ClampAndFinite(t *testing.T) {
	s := validSample()
	s.SpeedKmh = -1
	s.FuelLevelL = math.Inf(1)
	s.FuelRateLph = -3
	s.BrakePipePressureBar = -4
	s.MainReservoirBar = math.NaN()
	s.EngineOilPressureBar = -6
	s.CoolantTempC = -30
	s.EngineOilTempC = math.Inf(-1)
	s.BatteryVoltageV = -8
	s.TractionCurrentA = -20
	s.LineVoltageV = -22
	s.MileageKm = -100

	out, err := NormalizeSample(&s)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if out.SpeedKmh != 0 || out.FuelLevelL != 0 || out.FuelRateLph != 0 ||
		out.BrakePipePressureBar != 0 || out.MainReservoirBar != 0 ||
		out.EngineOilPressureBar != 0 || out.CoolantTempC != 0 ||
		out.EngineOilTempC != 0 || out.BatteryVoltageV != 0 ||
		out.TractionCurrentA != 0 || out.LineVoltageV != 0 || out.MileageKm != 0 {
		t.Fatalf("clamp/finite failed: %+v", *out)
	}
}

func TestNormalizeSample_InvalidLatLon(t *testing.T) {
	t.Run("lat", func(t *testing.T) {
		s := validSample()
		s.Lat = 100
		_, err := NormalizeSample(&s)
		if !errors.Is(err, ErrLatInvalid) {
			t.Fatalf("err=%v, want %v", err, ErrLatInvalid)
		}
	})

	t.Run("lon", func(t *testing.T) {
		s := validSample()
		s.Lon = -181
		_, err := NormalizeSample(&s)
		if !errors.Is(err, ErrLonInvalid) {
			t.Fatalf("err=%v, want %v", err, ErrLonInvalid)
		}
	})
}

func TestNormalizeSample_AlertsAndHealthCleanup(t *testing.T) {
	s := validSample()
	s.Alerts = []telemetry.Alert{
		{Code: " A1 ", Severity: "INFO", Text: " motor hot "},
		{Code: "A1", Severity: "info", Text: "motor hot"},
		{Code: "A2", Severity: "unknown", Text: "x"},
		{Code: "", Severity: " ", Text: " "},
	}
	s.HealthIndex = 88
	s.HealthGrade = "A"
	s.HealthTopFactors = []telemetry.Factor{{Factor: "x", Penalty: 1.2}}

	out, err := NormalizeSample(&s)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if len(out.Alerts) != 2 {
		t.Fatalf("alerts len=%d, want 2", len(out.Alerts))
	}
	if out.Alerts[0].Severity != "info" {
		t.Fatalf("severity[0]=%q, want info", out.Alerts[0].Severity)
	}
	if out.Alerts[1].Severity != "warn" {
		t.Fatalf("severity[1]=%q, want warn", out.Alerts[1].Severity)
	}
	if out.HealthIndex != 0 || out.HealthGrade != "" || out.HealthTopFactors != nil {
		t.Fatalf("health cleanup failed: %+v", *out)
	}
}

func TestNormalizeSample_TractionMotorSanitize(t *testing.T) {
	s := validSample()
	s.TractionMotorTempC = []float64{60, math.NaN(), -5, math.Inf(1), 73}

	out, err := NormalizeSample(&s)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if len(out.TractionMotorTempC) != 3 {
		t.Fatalf("len=%d, want 3", len(out.TractionMotorTempC))
	}
	if out.TractionMotorTempC[0] != 60 || out.TractionMotorTempC[1] != 0 || out.TractionMotorTempC[2] != 73 {
		t.Fatalf("traction motor sanitize failed: %v", out.TractionMotorTempC)
	}
}

func TestProcessor_DedupWindow(t *testing.T) {
	store := NewStore(15*time.Minute, 5)
	p := NewProcessor(Options{
		EnableSmoothing: false,
		EnableDedup:     true,
		DedupWindow:     1500 * time.Millisecond,
		EMAAlpha:        0.4,
		State:           store,
	})

	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	p.now = func() time.Time { return now }

	s1 := validSample()
	s1.TS = "2026-04-05T10:00:00Z"
	r1, err := p.Prepare(&s1)
	if err != nil {
		t.Fatalf("prepare #1: %v", err)
	}
	if r1.Decision != DecisionPublish {
		t.Fatalf("decision #1=%v, want publish", r1.Decision)
	}
	r1.Commit()

	now = now.Add(900 * time.Millisecond)
	s2 := validSample()
	s2.TS = "2026-04-05T10:00:01Z" // ts отличается, fingerprint должен считаться без ts
	r2, err := p.Prepare(&s2)
	if err != nil {
		t.Fatalf("prepare #2: %v", err)
	}
	if r2.Decision != DecisionSkipDuplicate {
		t.Fatalf("decision #2=%v, want skip duplicate", r2.Decision)
	}
	r2.Commit()

	now = now.Add(2 * time.Second)
	r3, err := p.Prepare(&s2)
	if err != nil {
		t.Fatalf("prepare #3: %v", err)
	}
	if r3.Decision != DecisionPublish {
		t.Fatalf("decision #3=%v, want publish outside dedup window", r3.Decision)
	}
}

func TestProcessor_EMA(t *testing.T) {
	store := NewStore(15*time.Minute, 5)
	p := NewProcessor(Options{
		EnableSmoothing: true,
		EnableDedup:     false,
		EMAAlpha:        0.5,
		State:           store,
	})

	now := time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	p.now = func() time.Time { return now }

	s1 := validSample()
	s1.SpeedKmh = 10
	s1.FuelRateLph = 10
	s1.TractionCurrentA = 100
	s1.LineVoltageV = 1000
	s1.TractionMotorTempC = []float64{50}

	r1, err := p.Prepare(&s1)
	if err != nil {
		t.Fatalf("prepare #1: %v", err)
	}
	if r1.Sample.SpeedKmh != 10 || r1.Sample.FuelRateLph != 10 || r1.Sample.TractionCurrentA != 100 || r1.Sample.LineVoltageV != 1000 {
		t.Fatalf("first sample should not be smoothed: %+v", *r1.Sample)
	}
	r1.Commit()

	now = now.Add(1 * time.Second)
	s2 := validSample()
	s2.SpeedKmh = 30
	s2.FuelRateLph = 30
	s2.TractionCurrentA = 300
	s2.LineVoltageV = 2000
	s2.TractionMotorTempC = []float64{70}

	r2, err := p.Prepare(&s2)
	if err != nil {
		t.Fatalf("prepare #2: %v", err)
	}

	if math.Abs(r2.Sample.SpeedKmh-20) > 1e-9 {
		t.Fatalf("speed=%v, want 20", r2.Sample.SpeedKmh)
	}
	if math.Abs(r2.Sample.FuelRateLph-20) > 1e-9 {
		t.Fatalf("fuel_rate=%v, want 20", r2.Sample.FuelRateLph)
	}
	if r2.Sample.TractionCurrentA != 200 {
		t.Fatalf("traction_current=%d, want 200", r2.Sample.TractionCurrentA)
	}
	if r2.Sample.LineVoltageV != 1500 {
		t.Fatalf("line_voltage=%d, want 1500", r2.Sample.LineVoltageV)
	}
	if len(r2.Sample.TractionMotorTempC) != 1 || math.Abs(r2.Sample.TractionMotorTempC[0]-60) > 1e-9 {
		t.Fatalf("traction_motor=%v, want [60]", r2.Sample.TractionMotorTempC)
	}
}

func TestStore_CleanupExpired(t *testing.T) {
	store := NewStore(1*time.Minute, 5)
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)

	store.Commit("fresh", TrainState{LastSeen: now.Add(-30 * time.Second)})
	store.Commit("stale", TrainState{LastSeen: now.Add(-2 * time.Minute)})

	removed := store.CleanupExpired(now)
	if removed != 1 {
		t.Fatalf("removed=%d, want 1", removed)
	}
	if store.size() != 1 {
		t.Fatalf("size=%d, want 1", store.size())
	}
}

func validSample() telemetry.Sample {
	return telemetry.Sample{
		TS:                   "2026-04-05T00:00:00+06:00",
		TrainID:              " LOC-1 ",
		SpeedKmh:             10,
		FuelLevelL:           100,
		FuelRateLph:          8,
		BrakePipePressureBar: 5,
		MainReservoirBar:     8,
		EngineOilPressureBar: 4,
		CoolantTempC:         85,
		EngineOilTempC:       90,
		TractionMotorTempC:   []float64{70, 71},
		BatteryVoltageV:      110,
		TractionCurrentA:     220,
		LineVoltageV:         2700,
		Lat:                  51.1,
		Lon:                  71.4,
		MileageKm:            1234.5,
	}
}

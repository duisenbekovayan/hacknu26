package normalizer

import (
	"errors"
	"math"
	"testing"
	"time"

	"hacknu/pkg/telemetry"
)

func TestNormalizeSample_emptyTrainID(t *testing.T) {
	s := validSample()
	s.TrainID = "   "

	_, err := NormalizeSample(&s)
	if !errors.Is(err, errTrainIDMissing) {
		t.Fatalf("err = %v, want %v", err, errTrainIDMissing)
	}
}

func TestNormalizeSample_emptyTS(t *testing.T) {
	s := validSample()
	s.TS = ""

	out, err := NormalizeSample(&s)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.TS == "" {
		t.Fatalf("TS should be set")
	}
	if _, err := time.Parse(time.RFC3339, out.TS); err != nil {
		t.Fatalf("TS is not RFC3339: %v", err)
	}
}

func TestNormalizeSample_invalidTS(t *testing.T) {
	s := validSample()
	s.TS = "not-a-date"

	_, err := NormalizeSample(&s)
	if !errors.Is(err, errTSInvalid) {
		t.Fatalf("err = %v, want %v", err, errTSInvalid)
	}
}

func TestNormalizeSample_negativeValuesClampedToZero(t *testing.T) {
	s := validSample()
	s.SpeedKmh = -1
	s.FuelLevelL = -2
	s.FuelRateLph = -3
	s.BrakePipePressureBar = -4
	s.MainReservoirBar = -5
	s.EngineOilPressureBar = -6
	s.BatteryVoltageV = -7
	s.TractionCurrentA = -8
	s.LineVoltageV = -9
	s.MileageKm = -10

	out, err := NormalizeSample(&s)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.SpeedKmh != 0 || out.FuelLevelL != 0 || out.FuelRateLph != 0 ||
		out.BrakePipePressureBar != 0 || out.MainReservoirBar != 0 ||
		out.EngineOilPressureBar != 0 || out.BatteryVoltageV != 0 ||
		out.TractionCurrentA != 0 || out.LineVoltageV != 0 || out.MileageKm != 0 {
		t.Fatalf("negative clamp failed: %+v", *out)
	}
}

func TestNormalizeSample_invalidLatLon(t *testing.T) {
	t.Run("lat", func(t *testing.T) {
		s := validSample()
		s.Lat = 91
		_, err := NormalizeSample(&s)
		if !errors.Is(err, errLatInvalid) {
			t.Fatalf("err = %v, want %v", err, errLatInvalid)
		}
	})

	t.Run("lon", func(t *testing.T) {
		s := validSample()
		s.Lon = -181
		_, err := NormalizeSample(&s)
		if !errors.Is(err, errLonInvalid) {
			t.Fatalf("err = %v, want %v", err, errLonInvalid)
		}
	})
}

func TestNormalizeSample_alertsDedupAndSeverityNormalization(t *testing.T) {
	s := validSample()
	s.Alerts = []telemetry.Alert{
		{Code: " A1 ", Severity: "INFO", Text: " Motor hot "},
		{Code: "A1", Severity: "info", Text: "Motor hot"},
		{Code: "A2", Severity: "UNKNOWN", Text: "X"},
		{Code: "", Severity: "   ", Text: "   "},
	}

	out, err := NormalizeSample(&s)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(out.Alerts) != 2 {
		t.Fatalf("alerts len = %d, want 2", len(out.Alerts))
	}
	if out.Alerts[0].Severity != "info" {
		t.Fatalf("severity #1 = %q, want info", out.Alerts[0].Severity)
	}
	if out.Alerts[1].Severity != "warn" {
		t.Fatalf("severity #2 = %q, want warn", out.Alerts[1].Severity)
	}
}

func TestNormalizeSample_healthFieldsCleared(t *testing.T) {
	s := validSample()
	s.HealthIndex = 77.7
	s.HealthGrade = "B"
	s.HealthTopFactors = []telemetry.Factor{{Factor: "x", Penalty: 1.2}}

	out, err := NormalizeSample(&s)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.HealthIndex != 0 {
		t.Fatalf("HealthIndex = %v, want 0", out.HealthIndex)
	}
	if out.HealthGrade != "" {
		t.Fatalf("HealthGrade = %q, want empty", out.HealthGrade)
	}
	if out.HealthTopFactors != nil {
		t.Fatalf("HealthTopFactors should be nil")
	}
}

func TestNormalizeSample_nanInfHandling(t *testing.T) {
	s := validSample()
	s.SpeedKmh = math.Inf(1)
	s.CoolantTempC = math.NaN()
	s.EngineOilTempC = math.Inf(-1)
	s.TractionMotorTempC = []float64{63.5, math.NaN(), math.Inf(1), 70.2}

	out, err := NormalizeSample(&s)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.SpeedKmh != 0 {
		t.Fatalf("SpeedKmh = %v, want 0", out.SpeedKmh)
	}
	if out.CoolantTempC != 0 {
		t.Fatalf("CoolantTempC = %v, want 0", out.CoolantTempC)
	}
	if out.EngineOilTempC != 0 {
		t.Fatalf("EngineOilTempC = %v, want 0", out.EngineOilTempC)
	}
	if len(out.TractionMotorTempC) != 2 || out.TractionMotorTempC[0] != 63.5 || out.TractionMotorTempC[1] != 70.2 {
		t.Fatalf("traction motors = %v, want [63.5 70.2]", out.TractionMotorTempC)
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

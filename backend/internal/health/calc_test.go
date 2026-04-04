package health

import (
	"testing"

	"hacknu/pkg/telemetry"
)

func TestApply_nominal(t *testing.T) {
	s := &telemetry.Sample{
		CoolantTempC:         90,
		EngineOilPressureBar: 4.5,
		BatteryVoltageV:      110,
		TractionMotorTempC:   []float64{70, 70, 70, 70, 70, 70},
		MainReservoirBar:     8.0,
		Alerts:               nil,
	}
	Apply(s)
	if s.HealthIndex != 100 {
		t.Fatalf("HealthIndex = %v, want 100", s.HealthIndex)
	}
	if s.HealthGrade != gradeA {
		t.Fatalf("grade = %s, want A", s.HealthGrade)
	}
}

func TestApply_coolantOver98(t *testing.T) {
	s := &telemetry.Sample{
		CoolantTempC:         100,
		EngineOilPressureBar: 4.5,
		BatteryVoltageV:      110,
		TractionMotorTempC:   []float64{70, 70, 70, 70, 70, 70},
		MainReservoirBar:     8.0,
	}
	Apply(s)
	// (100-98)*5 = 10 -> 90
	if s.HealthIndex != 90 {
		t.Fatalf("HealthIndex = %v, want 90", s.HealthIndex)
	}
}

func TestApply_lowOilPressure(t *testing.T) {
	s := &telemetry.Sample{
		CoolantTempC:         90,
		EngineOilPressureBar: 3.0,
		BatteryVoltageV:      110,
		TractionMotorTempC:   []float64{70, 70, 70, 70, 70, 70},
		MainReservoirBar:     8.0,
	}
	Apply(s)
	if s.HealthIndex != 85 {
		t.Fatalf("HealthIndex = %v, want 85", s.HealthIndex)
	}
	if s.HealthGrade != gradeA {
		t.Fatalf("grade = %s, want A (85 на границе нормы)", s.HealthGrade)
	}
}

func TestApply_motorOver115(t *testing.T) {
	s := &telemetry.Sample{
		CoolantTempC:         90,
		EngineOilPressureBar: 4.5,
		BatteryVoltageV:      110,
		TractionMotorTempC:   []float64{116, 70, 70, 70, 70, 70},
		MainReservoirBar:     8.0,
	}
	Apply(s)
	if s.HealthIndex != 92 {
		t.Fatalf("HealthIndex = %v, want 92", s.HealthIndex)
	}
}

func TestApply_alerts(t *testing.T) {
	s := &telemetry.Sample{
		CoolantTempC:         90,
		EngineOilPressureBar: 4.5,
		BatteryVoltageV:      110,
		TractionMotorTempC:   []float64{70, 70, 70, 70, 70, 70},
		MainReservoirBar:     8.0,
		Alerts: []telemetry.Alert{
			{Code: "X1", Severity: "warn", Text: "w"},
			{Code: "X2", Severity: "crit", Text: "c"},
		},
	}
	Apply(s)
	// 5 + 15 = 20 -> 80
	if s.HealthIndex != 80 {
		t.Fatalf("HealthIndex = %v, want 80", s.HealthIndex)
	}
}

func TestApply_topFactorsMax5(t *testing.T) {
	m := make([]float64, 6)
	for i := range m {
		m[i] = 120
	}
	s := &telemetry.Sample{
		CoolantTempC:         100,
		EngineOilPressureBar: 3.0,
		BatteryVoltageV:      95,
		TractionMotorTempC:   m,
		MainReservoirBar:     6.5,
		Alerts: []telemetry.Alert{
			{Code: "W", Severity: "warn", Text: ""},
		},
	}
	Apply(s)
	if len(s.HealthTopFactors) > 5 {
		t.Fatalf("len top = %d, want <= 5", len(s.HealthTopFactors))
	}
}

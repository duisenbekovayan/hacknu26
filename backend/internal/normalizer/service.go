package normalizer

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"hacknu/pkg/telemetry"
)

var (
	errNilSample      = errors.New("sample is nil")
	errTrainIDMissing = errors.New("train_id required")
	errTSInvalid      = errors.New("ts must be valid RFC3339")
	errLatInvalid     = errors.New("lat out of range [-90, 90]")
	errLonInvalid     = errors.New("lon out of range [-180, 180]")
)

// NormalizeSample валидирует и нормализует входящий сэмпл в канонический вид.
func NormalizeSample(in *telemetry.Sample) (*telemetry.Sample, error) {
	if in == nil {
		return nil, errNilSample
	}

	s := *in
	s.TrainID = strings.TrimSpace(s.TrainID)
	if s.TrainID == "" {
		return nil, errTrainIDMissing
	}

	ts, err := normalizeTS(s.TS)
	if err != nil {
		return nil, err
	}
	s.TS = ts

	s.SpeedKmh = nonNegativeOrZero(s.SpeedKmh)
	s.FuelLevelL = nonNegativeOrZero(s.FuelLevelL)
	s.FuelRateLph = nonNegativeOrZero(s.FuelRateLph)
	s.BrakePipePressureBar = nonNegativeOrZero(s.BrakePipePressureBar)
	s.MainReservoirBar = nonNegativeOrZero(s.MainReservoirBar)
	s.EngineOilPressureBar = nonNegativeOrZero(s.EngineOilPressureBar)
	s.CoolantTempC = finiteOrZero(s.CoolantTempC)
	s.EngineOilTempC = finiteOrZero(s.EngineOilTempC)
	s.BatteryVoltageV = nonNegativeOrZero(s.BatteryVoltageV)
	s.TractionCurrentA = nonNegativeInt(s.TractionCurrentA)
	s.LineVoltageV = nonNegativeInt(s.LineVoltageV)
	s.MileageKm = nonNegativeOrZero(s.MileageKm)

	if !isFinite(s.Lat) || s.Lat < -90 || s.Lat > 90 {
		return nil, errLatInvalid
	}
	if !isFinite(s.Lon) || s.Lon < -180 || s.Lon > 180 {
		return nil, errLonInvalid
	}

	s.TractionMotorTempC = finiteSlice(s.TractionMotorTempC)
	s.Alerts = normalizeAlerts(s.Alerts)

	// Health всегда пересчитывает backend.
	s.HealthIndex = 0
	s.HealthGrade = ""
	s.HealthTopFactors = nil

	return &s, nil
}

func normalizeTS(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Now().UTC().Format(time.RFC3339), nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errTSInvalid, err)
	}
	return parsed.UTC().Format(time.RFC3339), nil
}

func normalizeAlerts(in []telemetry.Alert) []telemetry.Alert {
	if len(in) == 0 {
		return nil
	}
	out := make([]telemetry.Alert, 0, len(in))
	seen := make(map[string]struct{}, len(in))

	for _, a := range in {
		code := strings.TrimSpace(a.Code)
		text := strings.TrimSpace(a.Text)
		rawSeverity := strings.TrimSpace(strings.ToLower(a.Severity))
		if code == "" && text == "" && rawSeverity == "" {
			continue
		}
		severity := normalizeSeverity(rawSeverity)
		key := code + "\x00" + severity + "\x00" + text
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, telemetry.Alert{
			Code:     code,
			Severity: severity,
			Text:     text,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeSeverity(v string) string {
	switch v {
	case "info", "warn", "crit":
		return v
	default:
		return "warn"
	}
}

func nonNegativeOrZero(v float64) float64 {
	if !isFinite(v) || v < 0 {
		return 0
	}
	return v
}

func finiteOrZero(v float64) float64 {
	if !isFinite(v) {
		return 0
	}
	return v
}

func nonNegativeInt(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func finiteSlice(in []float64) []float64 {
	if len(in) == 0 {
		return nil
	}
	out := make([]float64, 0, len(in))
	for _, v := range in {
		if isFinite(v) {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

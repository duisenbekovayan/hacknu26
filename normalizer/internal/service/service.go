package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"hacknu/pkg/telemetry"
)

var (
	ErrNilSample      = errors.New("sample is nil")
	ErrTrainIDMissing = errors.New("train_id required")
	ErrTSInvalid      = errors.New("ts must be valid RFC3339")
	ErrLatInvalid     = errors.New("lat out of range [-90, 90]")
	ErrLonInvalid     = errors.New("lon out of range [-180, 180]")
)

type Decision int

const (
	DecisionPublish Decision = iota
	DecisionSkipDuplicate
)

type Result struct {
	Decision Decision
	Sample   *telemetry.Sample
	TrainID  string
	Commit   func()
}

type Options struct {
	EnableSmoothing bool
	EnableDedup     bool
	DedupWindow     time.Duration
	EMAAlpha        float64
	State           *Store
}

// Processor выполняет preprocessing pipeline перед публикацией normalized-события.
type Processor struct {
	state           *Store
	enableSmoothing bool
	enableDedup     bool
	dedupWindow     time.Duration
	emaAlpha        float64
	now             func() time.Time
}

func NewProcessor(opts Options) *Processor {
	state := opts.State
	if state == nil {
		state = NewStore(15*time.Minute, 5)
	}

	dedupWindow := opts.DedupWindow
	if dedupWindow <= 0 {
		dedupWindow = 1500 * time.Millisecond
	}

	emaAlpha := opts.EMAAlpha
	if emaAlpha <= 0 || emaAlpha > 1 {
		emaAlpha = 0.4
	}

	return &Processor{
		state:           state,
		enableSmoothing: opts.EnableSmoothing,
		enableDedup:     opts.EnableDedup,
		dedupWindow:     dedupWindow,
		emaAlpha:        emaAlpha,
		now:             time.Now,
	}
}

func (p *Processor) State() *Store {
	return p.state
}

// Prepare валидирует/нормализует sample, применяет dedup/smoothing и возвращает результат.
// State обновляется только через Commit после успешной публикации downstream.
func (p *Processor) Prepare(in *telemetry.Sample) (Result, error) {
	norm, err := NormalizeSample(in)
	if err != nil {
		return Result{}, err
	}

	now := p.now().UTC()
	fingerprint, err := fingerprintWithoutTS(norm)
	if err != nil {
		return Result{}, fmt.Errorf("fingerprint: %w", err)
	}

	state := p.state.Snapshot(norm.TrainID)
	if p.enableDedup && state.LastFingerprint == fingerprint && now.Sub(state.LastFingerprintAt) <= p.dedupWindow {
		state.LastSeen = now
		committed := cloneTrainState(state)
		trainID := norm.TrainID
		return Result{
			Decision: DecisionSkipDuplicate,
			TrainID:  trainID,
			Commit: func() {
				p.state.Commit(trainID, committed)
			},
		}, nil
	}

	out := *norm
	if p.enableSmoothing {
		applyEMA(&out, &state.EMA, p.emaAlpha)
	}

	state.LastFingerprint = fingerprint
	state.LastFingerprintAt = now
	state.LastSeen = now
	state.Buffer = append(state.Buffer, out)
	if len(state.Buffer) > p.state.BufferSize() {
		state.Buffer = state.Buffer[len(state.Buffer)-p.state.BufferSize():]
	}

	committed := cloneTrainState(state)
	trainID := out.TrainID
	return Result{
		Decision: DecisionPublish,
		Sample:   &out,
		TrainID:  trainID,
		Commit: func() {
			p.state.Commit(trainID, committed)
		},
	}, nil
}

// NormalizeSample валидирует и нормализует входящий sample.
func NormalizeSample(in *telemetry.Sample) (*telemetry.Sample, error) {
	if in == nil {
		return nil, ErrNilSample
	}

	s := *in
	s.TrainID = strings.TrimSpace(s.TrainID)
	if s.TrainID == "" {
		return nil, ErrTrainIDMissing
	}

	ts, err := normalizeTS(s.TS)
	if err != nil {
		return nil, err
	}
	s.TS = ts

	s.SpeedKmh = nonNegativeFiniteOrZero(s.SpeedKmh)
	s.FuelLevelL = nonNegativeFiniteOrZero(s.FuelLevelL)
	s.FuelRateLph = nonNegativeFiniteOrZero(s.FuelRateLph)
	s.BrakePipePressureBar = nonNegativeFiniteOrZero(s.BrakePipePressureBar)
	s.MainReservoirBar = nonNegativeFiniteOrZero(s.MainReservoirBar)
	s.EngineOilPressureBar = nonNegativeFiniteOrZero(s.EngineOilPressureBar)
	s.CoolantTempC = nonNegativeFiniteOrZero(s.CoolantTempC)
	s.EngineOilTempC = nonNegativeFiniteOrZero(s.EngineOilTempC)
	s.BatteryVoltageV = nonNegativeFiniteOrZero(s.BatteryVoltageV)
	s.TractionCurrentA = nonNegativeInt(s.TractionCurrentA)
	s.LineVoltageV = nonNegativeInt(s.LineVoltageV)
	s.MileageKm = nonNegativeFiniteOrZero(s.MileageKm)

	if !isFinite(s.Lat) || s.Lat < -90 || s.Lat > 90 {
		return nil, ErrLatInvalid
	}
	if !isFinite(s.Lon) || s.Lon < -180 || s.Lon > 180 {
		return nil, ErrLonInvalid
	}

	s.TractionMotorTempC = sanitizeTractionMotorTemps(s.TractionMotorTempC)
	s.Alerts = normalizeAlerts(s.Alerts)

	// Health всегда пересчитывается backend-service.
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
		return "", fmt.Errorf("%w: %v", ErrTSInvalid, err)
	}
	return parsed.UTC().Format(time.RFC3339), nil
}

func sanitizeTractionMotorTemps(in []float64) []float64 {
	if len(in) == 0 {
		return nil
	}
	out := make([]float64, 0, len(in))
	for _, v := range in {
		if !isFinite(v) {
			continue
		}
		if v < 0 {
			out = append(out, 0)
			continue
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
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

func normalizeSeverity(s string) string {
	switch s {
	case "info", "warn", "crit":
		return s
	default:
		return "warn"
	}
}

func nonNegativeFiniteOrZero(v float64) float64 {
	if !isFinite(v) || v < 0 {
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

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func applyEMA(s *telemetry.Sample, ema *emaState, alpha float64) {
	s.SpeedKmh = ema.SpeedKmh.Apply(s.SpeedKmh, alpha)
	s.FuelRateLph = ema.FuelRateLph.Apply(s.FuelRateLph, alpha)
	s.BrakePipePressureBar = ema.BrakePipePressureBar.Apply(s.BrakePipePressureBar, alpha)
	s.MainReservoirBar = ema.MainReservoirBar.Apply(s.MainReservoirBar, alpha)
	s.EngineOilPressureBar = ema.EngineOilPressureBar.Apply(s.EngineOilPressureBar, alpha)
	s.CoolantTempC = ema.CoolantTempC.Apply(s.CoolantTempC, alpha)
	s.EngineOilTempC = ema.EngineOilTempC.Apply(s.EngineOilTempC, alpha)
	s.BatteryVoltageV = ema.BatteryVoltageV.Apply(s.BatteryVoltageV, alpha)

	tracA := ema.TractionCurrentA.Apply(float64(s.TractionCurrentA), alpha)
	s.TractionCurrentA = int(math.Round(tracA))

	lineV := ema.LineVoltageV.Apply(float64(s.LineVoltageV), alpha)
	s.LineVoltageV = int(math.Round(lineV))

	if len(s.TractionMotorTempC) == 0 {
		ema.TractionMotorTempC = nil
		return
	}
	if len(ema.TractionMotorTempC) < len(s.TractionMotorTempC) {
		ema.TractionMotorTempC = append(ema.TractionMotorTempC, make([]emaValue, len(s.TractionMotorTempC)-len(ema.TractionMotorTempC))...)
	}
	for i, v := range s.TractionMotorTempC {
		s.TractionMotorTempC[i] = ema.TractionMotorTempC[i].Apply(v, alpha)
	}
	if len(ema.TractionMotorTempC) > len(s.TractionMotorTempC) {
		ema.TractionMotorTempC = ema.TractionMotorTempC[:len(s.TractionMotorTempC)]
	}
}

func fingerprintWithoutTS(s *telemetry.Sample) (string, error) {
	copySample := *s
	copySample.TS = ""
	body, err := json.Marshal(copySample)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

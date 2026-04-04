package health

import (
	"math"
	"sort"
	"strconv"

	"hacknu/pkg/telemetry"
)

const (
	gradeA = "A"
	gradeB = "B"
	gradeC = "C"
	gradeD = "D"
	gradeE = "E"
)

// Apply пересчитывает индекс и grade; не мутирует слайсы алертов.
func Apply(s *telemetry.Sample) {
	pen := 0.0
	var factors []telemetry.Factor

	add := func(name string, v float64) {
		if v <= 0 {
			return
		}
		pen += v
		factors = append(factors, telemetry.Factor{Factor: name, Penalty: round1(v)})
	}

	if s.CoolantTempC > 98 {
		v := (s.CoolantTempC - 98) * 5
		add("перегрев ОЖ", v)
	}
	if s.EngineOilPressureBar < 3.2 {
		add("низкое давление масла", 15)
	}
	if s.BatteryVoltageV < 100 || s.BatteryVoltageV > 128 {
		add("напряжение АКБ", 10)
	}
	for i, t := range s.TractionMotorTempC {
		if t > 115 {
			add("ТЭД"+strconv.Itoa(i+1)+" перегрев", 8)
		}
	}
	if s.MainReservoirBar < 7.0 {
		add("низкое давление ГР", 12)
	}
	for _, a := range s.Alerts {
		switch a.Severity {
		case "crit":
			add("алерт "+a.Code, 15)
		case "warn":
			add("алерт "+a.Code, 5)
		}
	}

	hi := math.Max(0, math.Min(100, round1(100-pen)))
	s.HealthIndex = hi
	s.HealthGrade = gradeFrom(hi)

	sort.Slice(factors, func(i, j int) bool { return factors[i].Penalty > factors[j].Penalty })
	if len(factors) > 5 {
		factors = factors[:5]
	}
	s.HealthTopFactors = factors
}

func gradeFrom(hi float64) string {
	switch {
	case hi >= 85:
		return gradeA
	case hi >= 70:
		return gradeB
	case hi >= 60:
		return gradeC
	case hi >= 40:
		return gradeD
	default:
		return gradeE
	}
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}

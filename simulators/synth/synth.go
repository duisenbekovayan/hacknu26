package synth

import (
	"math"
	"math/rand"
	"time"

	"hacknu/pkg/telemetry"
)

// Synthesizer генерирует телеметрию из PRNG и внутреннего состояния (без файлов).
type Synthesizer struct {
	rng *rand.Rand
	// скорость и пробег
	speedKmh   float64
	mileageKm  float64
	fuelL      float64
	lat        float64
	lon        float64
	coolantC   float64
	engineOilC float64
	oilBar     float64
	brakeBar   float64
	mainBar    float64
	motors     [6]float64
}

// NewSynthesizer создаёт генератор с начальным состоянием от rng.
func NewSynthesizer(rng *rand.Rand) *Synthesizer {
	return &Synthesizer{
		rng:        rng,
		speedKmh:   rng.Float64() * 5,
		mileageKm:  rng.Float64() * 2,
		fuelL:      3800 + rng.Float64()*800,
		lat:        51.14 + rng.Float64()*0.02,
		lon:        71.42 + rng.Float64()*0.02,
		coolantC:   82 + rng.Float64()*6,
		engineOilC: 88 + rng.Float64()*6,
		oilBar:     4.2 + rng.Float64()*0.6,
		brakeBar:   4.95 + rng.Float64()*0.08,
		mainBar:    7.9 + rng.Float64()*0.25,
		motors:     randomMotors(rng, 66, 74),
	}
}

func randomMotors(rng *rand.Rand, lo, hi float64) [6]float64 {
	var m [6]float64
	for i := range m {
		m[i] = lo + rng.Float64()*(hi-lo)
	}
	return m
}

// NextSample формирует одну запись для trainID.
func (g *Synthesizer) NextSample(trainID string) telemetry.Sample {
	g.walkSpeed()
	g.mileageKm += g.speedKmh / 3600.0
	g.driftGeo()
	g.walkTempsAndPressure()
	g.fuelL -= g.fuelBurnPerSecond()
	if g.fuelL < 400 {
		g.fuelL = 3800 + g.rng.Float64()*400
	}

	spread := g.speedKmh * 0.12
	fuelRate := 6 + g.speedKmh*0.38 + (g.rng.Float64()-0.5)*1.2
	if fuelRate < 5 {
		fuelRate = 5
	}
	tracA := int(math.Round(math.Max(0, g.speedKmh*11+(g.rng.Float64()-0.5)*90)))
	lineV := int(math.Round(2720 + g.rng.Float64()*80))

	return telemetry.Sample{
		TS:                   time.Now().UTC().Format(time.RFC3339),
		TrainID:              trainID,
		SpeedKmh:             round1(g.speedKmh),
		FuelLevelL:           round1(g.fuelL),
		FuelRateLph:          round1(fuelRate),
		BrakePipePressureBar: clamp(g.brakeBar+(g.rng.Float64()-0.5)*0.04, 4.85, 5.15),
		MainReservoirBar:     clamp(g.mainBar+(g.rng.Float64()-0.5)*0.06, 7.5, 9.0),
		EngineOilPressureBar: clamp(g.oilBar+(g.rng.Float64()-0.5)*0.08, 3.4, 5.8),
		CoolantTempC:         clamp(g.coolantC+spread*0.15+(g.rng.Float64()-0.5)*0.5, 78, 108),
		EngineOilTempC:       clamp(g.engineOilC+g.speedKmh*0.06+(g.rng.Float64()-0.5)*0.4, 85, 118),
		TractionMotorTempC:   g.motorSlice(),
		BatteryVoltageV:      clamp(109+(g.rng.Float64()-0.5)*1.4, 102, 118),
		TractionCurrentA:     tracA,
		LineVoltageV:         lineV,
		Lat:                  round6(g.lat),
		Lon:                  round6(g.lon),
		MileageKm:            round3(g.mileageKm),
		Alerts:               nil,
	}
}

func (g *Synthesizer) walkSpeed() {
	switch g.rng.Intn(200) {
	case 0:
		g.speedKmh += 8 + g.rng.Float64()*12
	case 1:
		g.speedKmh -= 6 + g.rng.Float64()*10
	default:
		g.speedKmh += (g.rng.Float64() - 0.5) * 2.2
	}
	g.speedKmh = clamp(g.speedKmh, 0, 92)
}

func (g *Synthesizer) driftGeo() {
	g.lat += (g.rng.Float64() - 0.5) * 1.2e-5
	g.lon += g.speedKmh * 2e-8
}

func (g *Synthesizer) walkTempsAndPressure() {
	g.coolantC += (g.rng.Float64()-0.5)*0.35 + g.speedKmh*0.004
	g.engineOilC += (g.rng.Float64()-0.5)*0.25 + g.speedKmh*0.002
	g.oilBar += (g.rng.Float64() - 0.5) * 0.06
	g.brakeBar += (g.rng.Float64() - 0.5) * 0.02
	g.mainBar += (g.rng.Float64() - 0.5) * 0.03

	for i := range g.motors {
		g.motors[i] += (g.rng.Float64()-0.5)*0.6 + g.speedKmh*0.012
		g.motors[i] = clamp(g.motors[i], 60, 125)
	}

	g.coolantC = clamp(g.coolantC, 78, 105)
	g.engineOilC = clamp(g.engineOilC, 85, 115)
	g.oilBar = clamp(g.oilBar, 3.5, 5.5)
	g.brakeBar = clamp(g.brakeBar, 4.85, 5.2)
	g.mainBar = clamp(g.mainBar, 7.6, 9.0)
}

func (g *Synthesizer) fuelBurnPerSecond() float64 {
	rate := 6 + g.speedKmh*0.35 + g.rng.Float64()*0.3
	return rate / 3600.0
}

func (g *Synthesizer) motorSlice() []float64 {
	out := make([]float64, 6)
	for i := range g.motors {
		out[i] = round1(g.motors[i])
	}
	return out
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}

func round3(x float64) float64 {
	return math.Round(x*1000) / 1000
}

func round6(x float64) float64 {
	return math.Round(x*1e6) / 1e6
}

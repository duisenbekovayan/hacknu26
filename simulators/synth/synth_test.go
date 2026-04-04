package synth

import (
	"math/rand"
	"testing"
)

func TestNextSample_boundsAndMotors(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	g := NewSynthesizer(rng)
	for i := 0; i < 500; i++ {
		s := g.NextSample("TRAIN-X")
		if s.TrainID != "TRAIN-X" {
			t.Fatalf("train id")
		}
		if s.SpeedKmh < 0 || s.SpeedKmh > 92 {
			t.Fatalf("speed %v out of bounds", s.SpeedKmh)
		}
		if len(s.TractionMotorTempC) != 6 {
			t.Fatalf("motors len %d", len(s.TractionMotorTempC))
		}
		for _, tm := range s.TractionMotorTempC {
			if tm < 60 || tm > 125 {
				t.Fatalf("motor temp %v", tm)
			}
		}
		if s.BrakePipePressureBar < 4.85 || s.BrakePipePressureBar > 5.15 {
			t.Fatalf("brake pipe %v", s.BrakePipePressureBar)
		}
		if s.MainReservoirBar < 7.5 || s.MainReservoirBar > 9.0 {
			t.Fatalf("main %v", s.MainReservoirBar)
		}
		if s.FuelLevelL < 400 {
			t.Fatalf("fuel refuel broken %v", s.FuelLevelL)
		}
	}
}

func TestNewSynthesizer_deterministicSeed(t *testing.T) {
	r1 := rand.New(rand.NewSource(99))
	r2 := rand.New(rand.NewSource(99))
	g1 := NewSynthesizer(r1)
	g2 := NewSynthesizer(r2)
	a := g1.NextSample("T")
	b := g2.NextSample("T")
	if a.SpeedKmh != b.SpeedKmh || a.FuelLevelL != b.FuelLevelL {
		t.Fatalf("same seed should match first tick")
	}
}

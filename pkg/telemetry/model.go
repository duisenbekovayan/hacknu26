package telemetry

import "time"

// Alert соответствует мок-данным кейса.
type Alert struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // info | warn | crit
	Text     string `json:"text"`
}

// Sample — одна запись телеметрии (ingest / WS / история).
type Sample struct {
	TS                   string    `json:"ts"`
	TrainID              string    `json:"train_id"`
	SpeedKmh             float64   `json:"speed_kmh"`
	FuelLevelL           float64   `json:"fuel_level_l"`
	FuelRateLph          float64   `json:"fuel_rate_lph"`
	BrakePipePressureBar float64   `json:"brake_pipe_pressure_bar"`
	MainReservoirBar     float64   `json:"main_reservoir_bar"`
	EngineOilPressureBar float64   `json:"engine_oil_pressure_bar"`
	CoolantTempC         float64   `json:"coolant_temp_c"`
	EngineOilTempC       float64   `json:"engine_oil_temp_c"`
	TractionMotorTempC   []float64 `json:"traction_motor_temp_c"`
	BatteryVoltageV      float64   `json:"battery_voltage_v"`
	TractionCurrentA     int       `json:"traction_current_a"`
	LineVoltageV         int       `json:"line_voltage_v"`
	Lat                  float64   `json:"lat"`
	Lon                  float64   `json:"lon"`
	MileageKm            float64   `json:"mileage_km"`
	Alerts               []Alert   `json:"alerts"`
	HealthIndex          float64   `json:"health_index"`
	HealthGrade          string    `json:"health_grade"`
	HealthTopFactors     []Factor  `json:"health_top_factors"`
}

// Factor — вклад в штраф (объяснимость top-5).
type Factor struct {
	Factor  string  `json:"factor"`
	Penalty float64 `json:"penalty"`
}

// ParsedTime возвращает время записи для БД.
func (s Sample) ParsedTime() (time.Time, error) {
	return time.Parse(time.RFC3339, s.TS)
}

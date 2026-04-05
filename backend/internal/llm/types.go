package llm

// HealthFactor — тот же смысл, что health_top_factors на дашборде (штраф к индексу).
type HealthFactor struct {
	Factor  string  `json:"factor"`
	Penalty float64 `json:"penalty"`
}

// AnalyzeInput — сжатый снимок телеметрии для интерпретации (не сырой поток).
type AnalyzeInput struct {
	TrainID    string  `json:"train_id"`
	Timestamp  string  `json:"timestamp"`
	HealthIndex float64 `json:"health_index"`
	HealthGrade string  `json:"health_grade,omitempty"`
	// HealthTopFactors — обязательны для согласованности с формулой индекса на бэкенде.
	HealthTopFactors []HealthFactor `json:"health_top_factors,omitempty"`
	Speed            float64        `json:"speed"`
	FuelLevel        float64        `json:"fuel_level"`
	EngineTemp       float64        `json:"engine_temp,omitempty"` // охлаждающая жидкость (как раньше)
	CoolantTempC     float64        `json:"coolant_temp_c,omitempty"`
	EngineOilTempC   float64        `json:"engine_oil_temp_c,omitempty"`
	TractionMotorTempC []float64    `json:"traction_motor_temp_c,omitempty"`
	BrakePressure    float64        `json:"brake_pressure"`
	Voltage          float64        `json:"voltage"`
	Current          float64        `json:"current"`
	Alerts           []string       `json:"alerts"`
	RecentTrendNotes []string       `json:"recent_trend_notes,omitempty"`
	// Mode: пусто — общий разбор; "actions" — акцент на шагах машиниста.
	Mode string `json:"mode,omitempty"`
}

// AnalyzeOutput — структурированный ответ модели (только JSON из промпта).
type AnalyzeOutput struct {
	Summary         string   `json:"summary"`
	Severity        string   `json:"severity"` // normal | warning | critical
	ProbableCauses  []string `json:"probable_causes"`
	Recommendations []string `json:"recommendations"`
	AffectedMetrics []string `json:"affected_metrics"`
	NextRisk        string   `json:"next_risk,omitempty"`
}

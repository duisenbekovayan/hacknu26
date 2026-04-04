package llm

// AnalyzeInput — сжатый снимок телеметрии для интерпретации (не сырой поток).
type AnalyzeInput struct {
	TrainID          string   `json:"train_id"`
	Timestamp        string   `json:"timestamp"`
	HealthIndex      float64  `json:"health_index"`
	Speed            float64  `json:"speed"`
	FuelLevel        float64  `json:"fuel_level"`
	EngineTemp       float64  `json:"engine_temp"`
	BrakePressure    float64  `json:"brake_pressure"`
	Voltage          float64  `json:"voltage"`
	Current          float64  `json:"current"`
	Alerts           []string `json:"alerts"`
	RecentTrendNotes []string `json:"recent_trend_notes,omitempty"`
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

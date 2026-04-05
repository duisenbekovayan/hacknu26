package com.example.mobile.data

import com.google.gson.annotations.SerializedName

data class Alert(
    @SerializedName("code") val code: String,
    @SerializedName("severity") val severity: String,
    @SerializedName("text") val text: String
)

data class Factor(
    @SerializedName("factor") val factor: String,
    @SerializedName("penalty") val penalty: Double
)

data class TelemetrySample(
    @SerializedName("ts") val ts: String,
    @SerializedName("train_id") val trainID: String,
    @SerializedName("speed_kmh") val speedKmh: Double,
    @SerializedName("fuel_level_l") val fuelLevelL: Double,
    @SerializedName("fuel_rate_lph") val fuelRateLph: Double,
    @SerializedName("brake_pipe_pressure_bar") val brakePipePressureBar: Double,
    @SerializedName("main_reservoir_bar") val mainReservoirBar: Double,
    @SerializedName("engine_oil_pressure_bar") val engineOilPressureBar: Double,
    @SerializedName("coolant_temp_c") val coolantTempC: Double,
    @SerializedName("engine_oil_temp_c") val engineOilTempC: Double,
    @SerializedName("traction_motor_temp_c") val tractionMotorTempC: List<Double>,
    @SerializedName("battery_voltage_v") val batteryVoltageV: Double,
    @SerializedName("traction_current_a") val tractionCurrentA: Int,
    @SerializedName("line_voltage_v") val lineVoltageV: Int,
    @SerializedName("lat") val lat: Double,
    @SerializedName("lon") val lon: Double,
    @SerializedName("mileage_km") val mileageKm: Double,
    @SerializedName("alerts") val alerts: List<Alert>?,
    @SerializedName("health_index") val healthIndex: Double,
    @SerializedName("health_grade") val healthGrade: String,
    @SerializedName("health_top_factors") val healthTopFactors: List<Factor>?
)

data class AIRequest(
    @SerializedName("train_id") val trainId: String,
    @SerializedName("timestamp") val timestamp: String,
    @SerializedName("health_index") val healthIndex: Double,
    @SerializedName("health_grade") val healthGrade: String,
    @SerializedName("health_top_factors") val healthTopFactors: List<Factor>,
    @SerializedName("speed") val speed: Double,
    @SerializedName("fuel_level") val fuelLevel: Double,
    @SerializedName("engine_temp") val engineTemp: Double,
    @SerializedName("coolant_temp_c") val coolantTempC: Double,
    @SerializedName("engine_oil_temp_c") val engineOilTempC: Double,
    @SerializedName("traction_motor_temp_c") val tractionMotorTempC: List<Double>,
    @SerializedName("brake_pressure") val brakePressure: Double,
    @SerializedName("voltage") val voltage: Double,
    @SerializedName("current") val current: Int,
    @SerializedName("alerts") val alerts: List<String>,
    @SerializedName("mode") val mode: String? = null
)

data class AIResponse(
    @SerializedName("severity") val severity: String,
    @SerializedName("summary") val summary: String,
    @SerializedName("probable_causes") val probableCauses: List<String>,
    @SerializedName("recommendations") val recommendations: List<String>,
    @SerializedName("affected_metrics") val affectedMetrics: List<String>,
    @SerializedName("next_risk") val nextRisk: String?
)

package com.example.mobile.dashboard

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.mobile.api.LocomotiveApi
import com.example.mobile.data.AIRequest
import com.example.mobile.data.AIResponse
import com.example.mobile.data.TelemetrySample
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory

class DashboardViewModel : ViewModel() {

    private val _telemetry = MutableLiveData<TelemetrySample>()
    val telemetry: LiveData<TelemetrySample> = _telemetry

    private val _history = MutableLiveData<List<TelemetrySample>>()
    val history: LiveData<List<TelemetrySample>> = _history

    private val _aiResponse = MutableLiveData<AIResponse?>()
    val aiResponse: LiveData<AIResponse?> = _aiResponse

    private val _isAILoading = MutableLiveData<Boolean>(false)
    val isAILoading: LiveData<Boolean> = _isAILoading

    private val api = Retrofit.Builder()
        .baseUrl(com.example.mobile.BuildConfig.BASE_URL)
        .addConverterFactory(GsonConverterFactory.create())
        .build()
        .create(LocomotiveApi::class.java)

    fun startPolling(trainId: String) {
        viewModelScope.launch {
            while (true) {
                try {
                    _telemetry.postValue(api.getLatestTelemetry(trainId))
                    _history.postValue(api.getHistory(trainId, minutes = 15))
                } catch (e: Exception) {
                    e.printStackTrace()
                }
                delay(2000)
            }
        }
    }

    fun analyzeWithAI(mode: String? = null) {
        val currentTelemetry = _telemetry.value ?: return
        viewModelScope.launch {
            _isAILoading.postValue(true)
            try {
                val request = AIRequest(
                    trainId = currentTelemetry.trainID,
                    timestamp = currentTelemetry.ts,
                    healthIndex = currentTelemetry.healthIndex,
                    healthGrade = currentTelemetry.healthGrade,
                    healthTopFactors = currentTelemetry.healthTopFactors ?: emptyList(),
                    speed = currentTelemetry.speedKmh,
                    fuelLevel = currentTelemetry.fuelLevelL,
                    engineTemp = currentTelemetry.coolantTempC,
                    coolantTempC = currentTelemetry.coolantTempC,
                    engineOilTempC = currentTelemetry.engineOilTempC,
                    tractionMotorTempC = currentTelemetry.tractionMotorTempC,
                    brakePressure = currentTelemetry.brakePipePressureBar,
                    voltage = currentTelemetry.batteryVoltageV,
                    current = currentTelemetry.tractionCurrentA,
                    alerts = currentTelemetry.alerts?.map { (if (it.code.isNotEmpty()) "[${it.code}] " else "") + it.text } ?: emptyList(),
                    mode = mode
                )
                val response = api.analyzeWithAI(request)
                _aiResponse.postValue(response)
            } catch (e: Exception) {
                e.printStackTrace()
                // Optionally handle error
            } finally {
                _isAILoading.postValue(false)
            }
        }
    }

    fun clearAIResponse() {
        _aiResponse.value = null
    }
}

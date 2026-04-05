package com.example.mobile.dashboard

import android.util.Log
import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.mobile.BuildConfig
import com.example.mobile.api.LocomotiveApi
import com.example.mobile.data.AIRequest
import com.example.mobile.data.AIResponse
import com.example.mobile.data.TelemetrySample
import com.google.gson.Gson
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

class DashboardViewModel : ViewModel() {

    private val _telemetry = MutableLiveData<TelemetrySample>()
    val telemetry: LiveData<TelemetrySample> = _telemetry

    private val _history = MutableLiveData<List<TelemetrySample>>()
    val history: LiveData<List<TelemetrySample>> = _history

    private val _aiResponse = MutableLiveData<AIResponse?>()
    val aiResponse: LiveData<AIResponse?> = _aiResponse

    private val _isAILoading = MutableLiveData<Boolean>(false)
    val isAILoading: LiveData<Boolean> = _isAILoading

    private val _isDarkTheme = MutableLiveData<Boolean?>(null)
    val isDarkTheme: LiveData<Boolean?> = _isDarkTheme

    private val api = Retrofit.Builder()
        .baseUrl(BuildConfig.BASE_URL)
        .addConverterFactory(GsonConverterFactory.create())
        .build()
        .create(LocomotiveApi::class.java)

    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    fun startPolling(trainId: String) {
        viewModelScope.launch {
            while (true) {
                try {
                    val latest = api.getLatestTelemetry(trainId)
                    _telemetry.postValue(latest)
                    val historical = api.getHistory(trainId, minutes = 15)
                    _history.postValue(historical)
                } catch (e: Exception) {
                    Log.e("DashboardViewModel", "Polling error", e)
                }
                delay(2000)
            }
        }
    }

    fun analyzeWithAI(mode: String? = null, telemetry: TelemetrySample? = null) {
        val currentTelemetry = telemetry ?: _telemetry.value ?: return
        viewModelScope.launch {
            _isAILoading.postValue(true)
            
            val openaiKey = BuildConfig.OPENAI_API_KEY
            
            if (openaiKey.isNotEmpty()) {
                try {
                    Log.d("AIHelper", "Attempting OpenAI...")
                    val response = callOpenAI(currentTelemetry, mode, openaiKey)
                    _aiResponse.postValue(response)
                    _isAILoading.postValue(false)
                    return@launch
                } catch (e: Exception) {
                    Log.e("AIHelper", "OpenAI failed", e)
                }
            }

            Log.w("AIHelper", "OpenAI failed or no key. Falling back to backend AI.")
            try {
                val request = createAIRequest(currentTelemetry, mode)
                val response = api.analyzeWithAI(request)
                _aiResponse.postValue(response)
            } catch (e: Exception) {
                Log.e("AIHelper", "Backend AI error", e)
                _aiResponse.postValue(AIResponse(
                    severity = "warning",
                    summary = "Ошибка связи с ИИ",
                    probableCauses = listOf("Сервис временно недоступен", "Проблемы с сетью"),
                    recommendations = listOf("Попробуйте позже", "Проверьте подключение"),
                    affectedMetrics = emptyList(),
                    nextRisk = "Временная недоступность аналитики"
                ))
            } finally {
                _isAILoading.postValue(false)
            }
        }
    }

    private suspend fun callOpenAI(t: TelemetrySample, mode: String?, apiKey: String): AIResponse {
        val baseUrl = BuildConfig.OPENAI_BASE_URL
        val model = BuildConfig.OPENAI_MODEL
        val url = if (baseUrl.endsWith("/")) "${baseUrl}chat/completions" else "$baseUrl/chat/completions"
        
        val prompt = buildAiPrompt(t, mode)
        
        val jsonRequest = """
            {
              "model": "$model",
              "messages": [
                {"role": "system", "content": "Ты — помощник машиниста локомотива. Отвечай строго в формате JSON."},
                {"role": "user", "content": ${Gson().toJson(prompt)}}
              ],
              "response_format": { "type": "json_object" }
            }
        """.trimIndent()

        val requestBody = jsonRequest.toRequestBody("application/json".toMediaType())
        val request = Request.Builder()
            .url(url)
            .header("Authorization", "Bearer $apiKey")
            .post(requestBody)
            .build()

        val response = kotlinx.coroutines.withContext(kotlinx.coroutines.Dispatchers.IO) {
            client.newCall(request).execute()
        }
        
        val responseBody = response.body?.string() ?: ""
        if (!response.isSuccessful) {
            Log.e("AIHelper", "OpenAI error: ${response.code} - $responseBody")
            throw Exception("OpenAI error ${response.code}")
        }
        
        val openAiResponse = Gson().fromJson(responseBody, OpenAIRawResponse::class.java)
        val jsonText = openAiResponse.choices.firstOrNull()?.message?.content ?: ""
        
        return Gson().fromJson(extractJson(jsonText), AIResponse::class.java)
    }

    private fun extractJson(text: String): String {
        val startIndex = text.indexOf("{")
        val endIndex = text.lastIndexOf("}")
        if (startIndex == -1 || endIndex == -1) return text
        return text.substring(startIndex, endIndex + 1)
    }

    private fun createAIRequest(currentTelemetry: TelemetrySample, mode: String?): AIRequest {
        return AIRequest(
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
    }

    private fun buildAiPrompt(t: TelemetrySample, mode: String?): String {
        val context = """
            Локомотив: ${t.trainID}
            Индекс здоровья: ${t.healthIndex} (${t.healthGrade})
            Факторы риска: ${t.healthTopFactors?.joinToString { "${it.factor}: -${it.penalty}" }}
            Скорость: ${t.speedKmh} км/ч
            Температуры ТЭД: ${t.tractionMotorTempC.joinToString()}
            Давление тормозное: ${t.brakePipePressureBar} бар
            Напряжение: ${t.batteryVoltageV} В
            Ток: ${t.tractionCurrentA} А
            Алерты: ${t.alerts?.joinToString { it.text }}
            Режим запроса: ${mode ?: "Общий анализ"}
        """.trimIndent()

        return "Проанализируй данные телеметрии и верни JSON объект со следующими полями: " +
                "severity (critical, warning, normal), summary (краткий вывод), probable_causes (список строк), " +
                "recommendations (список строк), affected_metrics (список строк), next_risk (строка). " +
                "Данные: $context. Отвечай на русском языке. Верни ТОЛЬКО JSON."
    }

    fun clearAIResponse() {
        _aiResponse.value = null
    }

    fun toggleTheme() {
        val current = _isDarkTheme.value
        _isDarkTheme.value = if (current == true) false else true
    }
}

data class OpenAIRawResponse(val choices: List<OpenAIChoice>)
data class OpenAIChoice(val message: OpenAIMessage)
data class OpenAIMessage(val content: String)

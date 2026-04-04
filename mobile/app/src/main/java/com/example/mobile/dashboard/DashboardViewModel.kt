package com.example.mobile.dashboard

import androidx.lifecycle.LiveData
import androidx.lifecycle.MutableLiveData
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.mobile.api.LocomotiveApi
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
                    _history.postValue(api.getHistory(trainId, limit = 50))
                } catch (e: Exception) {
                    e.printStackTrace()
                }
                delay(2000)
            }
        }
    }
}

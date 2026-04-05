package com.example.mobile.api

import com.example.mobile.data.AIRequest
import com.example.mobile.data.AIResponse
import com.example.mobile.data.TelemetrySample
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Query

interface LocomotiveApi {
    @GET("api/v1/telemetry/latest")
    suspend fun getLatestTelemetry(
        @Query("train_id") trainId: String
    ): TelemetrySample

    @GET("api/v1/telemetry/history")
    suspend fun getHistory(
        @Query("train_id") trainId: String,
        @Query("minutes") minutes: Int = 15,
        @Query("limit") limit: Int = 500
    ): List<TelemetrySample>

    @POST("api/v1/ai/analyze")
    suspend fun analyzeWithAI(
        @Body request: AIRequest
    ): AIResponse
}

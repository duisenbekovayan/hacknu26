package com.example.mobile.api

import com.example.mobile.data.TelemetrySample
import retrofit2.http.GET
import retrofit2.http.Query

interface LocomotiveApi {
    @GET("api/v1/telemetry/latest")
    suspend fun getLatestTelemetry(
        @Query("train_id") trainId: String
    ): TelemetrySample

    @GET("api/v1/telemetry/history")
    suspend fun getHistory(
        @Query("train_id") trainId: String,
        @Query("limit") limit: Int = 500
    ): List<TelemetrySample>
}

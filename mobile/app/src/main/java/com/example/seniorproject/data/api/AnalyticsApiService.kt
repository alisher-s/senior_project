package com.example.seniorproject.data.api

import com.example.seniorproject.data.model.EventStatsResponseDTO
import retrofit2.Response
import retrofit2.http.GET
import retrofit2.http.Query

interface AnalyticsApiService {
    @GET("api/v1/analytics/events/stats")
    suspend fun getEventStats(
        @Query("event_id") eventId: String? = null
    ): Response<EventStatsResponseDTO>
}

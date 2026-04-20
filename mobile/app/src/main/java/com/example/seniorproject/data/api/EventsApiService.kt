package com.example.seniorproject.data.api

import com.example.seniorproject.data.model.CreateEventRequestDTO
import com.example.seniorproject.data.model.EventDTO
import com.example.seniorproject.data.model.ListEventsResponseDTO
import com.example.seniorproject.data.model.UpdateEventRequestDTO
import okhttp3.MultipartBody
import retrofit2.Response
import retrofit2.http.*

interface EventsApiService {
    @GET("api/v1/events")
    suspend fun listEvents(
        @Query("q") query: String? = null,
        @Query("limit") limit: Int = 20,
        @Query("offset") offset: Int = 0,
        @Query("starts_after") startsAfter: String? = null,
        @Query("starts_before") startsBefore: String? = null,
        @Query("organizer_id") organizerId: String? = null,
        @Query("moderation_status") moderationStatus: String? = null
    ): Response<ListEventsResponseDTO>

    @GET("api/v1/events/mine")
    suspend fun listMyEvents(
        @Query("organizer_id") organizerId: String? = null,
        @Query("limit") limit: Int = 20,
        @Query("offset") offset: Int = 0
    ): Response<ListEventsResponseDTO>

    @GET("api/v1/events/{id}")
    suspend fun getEvent(@Path("id") id: String): Response<EventDTO>

    @POST("api/v1/events")
    suspend fun createEvent(@Body request: CreateEventRequestDTO): Response<EventDTO>

    @PUT("api/v1/events/{id}")
    suspend fun updateEvent(
        @Path("id") id: String,
        @Body request: UpdateEventRequestDTO
    ): Response<EventDTO>

    @DELETE("api/v1/events/{id}")
    suspend fun deleteEvent(@Path("id") id: String): Response<Unit>

    @Multipart
    @POST("api/v1/events/{id}/cover-image")
    suspend fun uploadCoverImage(
        @Path("id") id: String,
        @Part image: MultipartBody.Part
    ): Response<Unit>
}

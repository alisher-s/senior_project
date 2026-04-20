package com.example.seniorproject.data.model

import com.google.gson.annotations.SerializedName

data class EventDTO(
    val id: String,
    val title: String,
    val description: String,
    @SerializedName("cover_image_url") val coverImageURL: String? = null,
    @SerializedName("starts_at") val startsAt: String,
    val location: String? = null,
    @SerializedName("end_at") val endAt: String? = null,
    @SerializedName("capacity_total") val capacityTotal: Int,
    @SerializedName("capacity_available") val capacityAvailable: Int,
    val status: String,
    @SerializedName("moderation_status") val moderationStatus: String,
    @SerializedName("organizer_id") val organizerId: String? = null
)

data class ListEventsResponseDTO(
    val items: List<EventDTO>,
    val limit: Int,
    val offset: Int
)

data class CreateEventRequestDTO(
    val title: String,
    val description: String,
    @SerializedName("cover_image_url") val coverImageURL: String? = null,
    @SerializedName("starts_at") val startsAt: String,
    val location: String? = null,
    @SerializedName("end_at") val endAt: String? = null,
    @SerializedName("capacity_total") val capacityTotal: Int
)

data class UpdateEventRequestDTO(
    val id: String? = null,
    val title: String? = null,
    val description: String? = null,
    @SerializedName("cover_image_url") val coverImageURL: String? = null,
    @SerializedName("starts_at") val startsAt: String? = null,
    val location: String? = null,
    @SerializedName("end_at") val endAt: String? = null,
    @SerializedName("capacity_total") val capacityTotal: Int? = null
)

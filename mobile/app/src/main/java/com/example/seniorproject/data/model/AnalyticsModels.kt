package com.example.seniorproject.data.model

import com.google.gson.annotations.SerializedName

data class RegistrationHour(
    val hour: String,
    val count: Int
)

data class EventStatsResponseDTO(
    @SerializedName("event_id") val eventId: String? = null,
    @SerializedName("total_capacity") val totalCapacity: Long,
    @SerializedName("registered_count") val registeredCount: Long,
    @SerializedName("remaining_capacity") val remainingCapacity: Long,
    @SerializedName("registration_timeline") val registrationTimeline: List<RegistrationHour>,
    @SerializedName("as_of") val asOf: String
)

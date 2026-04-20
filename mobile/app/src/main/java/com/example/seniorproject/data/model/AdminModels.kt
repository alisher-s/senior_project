package com.example.seniorproject.data.model

import com.google.gson.annotations.SerializedName

data class ModerateEventRequestDTO(
    val action: String, // "approve" or "reject"
    val reason: String? = null
)

data class ModerateEventResponseDTO(
    @SerializedName("moderation_status") val moderationStatus: String
)

data class SetUserRoleRequestDTO(
    val role: String // "student", "organizer", "admin"
)

data class UserRoleResponseDTO(
    val id: String,
    val email: String,
    val role: String
)

data class ListUsersResponseDTO(
    val items: List<UserDTO>,
    val limit: Int,
    val offset: Int
)

data class ModerationLogEntryDTO(
    val id: String,
    @SerializedName("admin_user_id") val adminUserID: String,
    @SerializedName("event_id") val eventID: String? = null,
    val action: String,
    val reason: String? = null,
    @SerializedName("created_at") val createdAt: String
)

data class ModerationLogsResponseDTO(
    val items: List<ModerationLogEntryDTO>,
    val limit: Int,
    val offset: Int
)

data class AdminEventDTO(
    val id: String,
    val title: String,
    @SerializedName("starts_at") val startsAt: String,
    @SerializedName("moderation_status") val moderationStatus: String,
    @SerializedName("organizer_id") val organizerId: String
)

data class ListAdminEventsResponseDTO(
    val items: List<AdminEventDTO>,
    val limit: Int,
    val offset: Int
)

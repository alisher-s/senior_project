package com.example.seniorproject.data.api

import com.example.seniorproject.data.model.ListUsersResponseDTO
import com.example.seniorproject.data.model.ModerateEventRequestDTO
import com.example.seniorproject.data.model.ModerateEventResponseDTO
import com.example.seniorproject.data.model.ModerationLogsResponseDTO
import com.example.seniorproject.data.model.SetUserRoleRequestDTO
import com.example.seniorproject.data.model.UserRoleResponseDTO
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

interface AdminApiService {
    @GET("api/v1/admin/users")
    suspend fun listUsers(
        @Query("q") query: String? = null,
        @Query("limit") limit: Int = 20,
        @Query("offset") offset: Int = 0
    ): Response<ListUsersResponseDTO>

    @POST("api/v1/admin/events/{id}/moderate")
    suspend fun moderateEvent(
        @Path("id") id: String,
        @Body request: ModerateEventRequestDTO
    ): Response<ModerateEventResponseDTO>

    @PATCH("api/v1/admin/users/{id}/role")
    suspend fun setUserRole(
        @Path("id") id: String,
        @Body request: SetUserRoleRequestDTO
    ): Response<UserRoleResponseDTO>

    @GET("api/v1/admin/moderation-logs")
    suspend fun listModerationLogs(
        @Query("event_id") eventID: String? = null,
        @Query("admin_id") adminID: String? = null,
        @Query("limit") limit: Int = 20,
        @Query("offset") offset: Int = 0
    ): Response<ModerationLogsResponseDTO>

    @GET("api/v1/admin/events")
    suspend fun listEvents(
        @Query("moderation_status") moderationStatus: String? = "pending",
        @Query("q") query: String? = null,
        @Query("limit") limit: Int = 20,
        @Query("offset") offset: Int = 0
    ): Response<com.example.seniorproject.data.model.ListAdminEventsResponseDTO>
}

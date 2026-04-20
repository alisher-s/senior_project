package com.example.seniorproject.data.model

import com.google.gson.annotations.SerializedName

data class UserDTO(
    val id: String,
    val email: String,
    val role: String,
    val roles: List<String>? = null,
    @SerializedName("pending_roles") val pendingRoles: List<String>? = null
)

data class PatchMeRolesRequestDTO(
    val roles: List<String>
)

data class MeRolesResponseDTO(
    val user: UserDTO
)

data class AuthResponseDTO(
    @SerializedName("access_token") val accessToken: String,
    @SerializedName("refresh_token") val refreshToken: String,
    val user: UserDTO
)

data class RegisterRequestDTO(
    val email: String,
    val password: String
)

data class LoginRequestDTO(
    val email: String,
    val password: String
)

data class RefreshRequestDTO(
    @SerializedName("refresh_token") val refreshToken: String
)
